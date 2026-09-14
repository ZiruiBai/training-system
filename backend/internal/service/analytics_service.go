package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// AnalyticsService handles assessments, reports, dashboard and risks.
type AnalyticsService struct {
	db *gorm.DB
}

// NewAnalyticsService creates an AnalyticsService.
func NewAnalyticsService(db *gorm.DB) *AnalyticsService { return &AnalyticsService{db: db} }

// --- Dashboard ---

// Dashboard returns role-aware metrics.
func (s *AnalyticsService) Dashboard(ctx context.Context, principal *model.Principal) (*model.DashboardOverview, error) {
	o := &model.DashboardOverview{}
	if principal.IsRole(model.RoleEmployee) && principal.EmployeeID != nil {
		return s.employeeDashboard(ctx, *principal.EmployeeID)
	}
	// Admin and manager both see org-level metrics. Manager could be restricted,
	// but for a single-tenant training platform the org overview is acceptable.
	var empTotal, empTraining, plansTotal, plansComplete int64
	s.db.WithContext(ctx).Model(&model.Employee{}).Count(&empTotal)
	s.db.WithContext(ctx).Model(&model.Employee{}).Where("training_status IN ?", []string{
		string(model.StatusInProgress), string(model.StatusNotStarted)}).Count(&empTraining)
	s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Count(&plansTotal)
	s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("status = ?", model.PlanCompleted).Count(&plansComplete)

	o.EmployeesTotal = int(empTotal)
	o.EmployeesInTraining = int(empTraining)
	o.PlansTotal = int(plansTotal)
	if plansTotal > 0 {
		o.PlanCompletionRate = float64(plansComplete) / float64(plansTotal) * 100
	}

	// Open tasks (pending vs completed across the org). Due-date-window
	// counting reads as 0/0 whenever nothing is due exactly today, which is
	// confusing; open-task counts are always meaningful.
	var totalTasks, doneTasks int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Count(&totalTasks)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).
		Where("status = ?", model.TaskCompleted).Count(&doneTasks)
	o.TasksToday = int(totalTasks - doneTasks)
	o.TasksTodayCompleted = int(doneTasks)
	if totalTasks > 0 {
		o.TasksTodayRate = float64(doneTasks) / float64(totalTasks) * 100
	}

	var avg float64
	s.db.WithContext(ctx).Model(&model.Employee{}).Select("COALESCE(AVG(progress),0)").Scan(&avg)
	o.AvgProgress = avg
	return o, nil
}

func (s *AnalyticsService) employeeDashboard(ctx context.Context, employeeID uint) (*model.DashboardOverview, error) {
	o := &model.DashboardOverview{EmployeesTotal: 1}
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("created_at desc").First(&plan).Error; err == nil {
		// Open tasks for this employee (pending vs completed) — always
		// meaningful, unlike a strict due-today window that reads 0/0.
		var total, done int64
		s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ?", employeeID).Count(&total)
		s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskCompleted).Count(&done)
		o.TasksToday = int(total - done)
		o.TasksTodayCompleted = int(done)
		if total > 0 {
			o.TasksTodayRate = float64(done) / float64(total) * 100
		}
		o.PlansTotal = 1
		o.AvgProgress = plan.Progress
	}
	return o, nil
}

// EmployeeProgressRows builds the admin "personnel training progress" table.
func (s *AnalyticsService) EmployeeProgressRows(ctx context.Context) ([]model.EmployeeProgressRow, error) {
	var emps []model.Employee
	if err := s.db.WithContext(ctx).Preload("Department").Preload("Position").
		Order("hire_date asc").Find(&emps).Error; err != nil {
		return nil, err
	}
	rows := make([]model.EmployeeProgressRow, 0, len(emps))
	for _, e := range emps {
		row := model.EmployeeProgressRow{
			EmployeeID: e.ID, Name: e.Name, CurrentStage: e.CurrentStage,
			Progress: e.Progress, TrainingState: string(e.TrainingStatus),
			TrainingDays: int(time.Since(e.HireDate).Hours() / 24),
		}
		if e.Position != nil {
			row.Position = e.Position.Name
		}
		if e.Department != nil {
			row.Department = e.Department.Name
		}
		row.RiskStatus = s.employeeRiskLevel(ctx, e.ID)
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *AnalyticsService) employeeRiskLevel(ctx context.Context, employeeID uint) string {
	var taskOverdue, taskPending int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).
		Where("employee_id = ? AND status = ?", employeeID, model.TaskOverdue).Count(&taskOverdue)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).
		Where("employee_id = ? AND status = ? AND due_date < ?", employeeID, model.TaskPending, time.Now().UTC()).Count(&taskPending)
	if taskOverdue >= 3 || taskPending >= 3 {
		return "high"
	}
	if taskOverdue > 0 || taskPending > 0 {
		return "medium"
	}
	return "low"
}

// DepartmentTrainingRows aggregates training by department.
func (s *AnalyticsService) DepartmentTrainingRows(ctx context.Context) ([]model.DepartmentTrainingRow, error) {
	var depts []model.Department
	if err := s.db.WithContext(ctx).Find(&depts).Error; err != nil {
		return nil, err
	}
	rows := []model.DepartmentTrainingRow{}
	for _, d := range depts {
		var emps []model.Employee
		s.db.WithContext(ctx).Where("department_id = ?", d.ID).Find(&emps)
		row := model.DepartmentTrainingRow{DepartmentID: d.ID, Department: d.Name}
		if len(emps) > 0 {
			var sum float64
			inTraining := 0
			for _, e := range emps {
				sum += e.Progress
				if e.TrainingStatus == model.StatusInProgress || e.TrainingStatus == model.StatusNotStarted {
					inTraining++
				}
				if s.employeeRiskLevel(ctx, e.ID) != "low" {
					row.RiskEmployees++
				}
			}
			row.InTraining = inTraining
			row.AvgProgress = sum / float64(len(emps))
			row.TaskRate = s.departmentTaskRate(ctx, d.ID)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *AnalyticsService) departmentTaskRate(ctx context.Context, departmentID uint) float64 {
	sub := s.db.Model(&model.Employee{}).Select("id").Where("department_id = ?", departmentID)
	var total, done int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id IN (?)", sub).Count(&total)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).
		Where("employee_id IN (?) AND status = ?", sub, model.TaskCompleted).Count(&done)
	if total == 0 {
		return 0
	}
	return float64(done) / float64(total) * 100
}

// Risks returns prioritized flag items.
func (s *AnalyticsService) Risks(ctx context.Context) ([]model.RiskItem, error) {
	var emps []model.Employee
	if err := s.db.WithContext(ctx).Preload("Department").Preload("Position").Find(&emps).Error; err != nil {
		return nil, err
	}
	risks := []model.RiskItem{}
	for _, e := range emps {
		base := model.RiskItem{EmployeeID: e.ID, Employee: e.Name}
		if e.Department != nil {
			base.Department = e.Department.Name
		}
		if e.Position != nil {
			base.Position = e.Position.Name
		}
		if e.TrainingStatus == model.StatusNotStarted {
			base.RiskType = "no_plan"
			base.Severity = "high"
			base.Message = "No training plan has been created yet"
			risks = append(risks, base)
			continue
		}
		if e.TrainingStatus == model.StatusPaused {
			base.RiskType = "paused"
			base.Severity = "high"
			base.Message = "Training plan is paused"
			risks = append(risks, base)
			continue
		}
		var overdue int64
		s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", e.ID, model.TaskOverdue).Count(&overdue)
		if overdue >= 3 {
			base.RiskType = "consecutive_overdue"
			base.Severity = "high"
			base.Message = "3+ overdue tasks in a row"
			risks = append(risks, base)
			continue
		}
		if overdue > 0 {
			base.RiskType = "overdue"
			base.Severity = "medium"
			base.Message = "Has overdue tasks"
			risks = append(risks, base)
		}
		if e.Progress < 25 && e.CurrentStage >= 2 {
			base.RiskType = "lagging"
			base.Severity = "medium"
			base.Message = "Learning progress is significantly behind"
			risks = append(risks, base)
		}
	}
	return risks, nil
}

// --- Assessments ---

// ListAssessments returns assessments with filters.
func (s *AnalyticsService) ListAssessments(ctx context.Context, opt ListOptions, employeeID, planID uint) ([]model.Assessment, int64, error) {
	var items []model.Assessment
	query := s.db.WithContext(ctx).Model(&model.Assessment{})
	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if planID > 0 {
		query = query.Where("plan_id = ?", planID)
	}
	if len(opt.EmployeeIDs) > 0 {
		query = query.Where("employee_id IN ?", opt.EmployeeIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Employee.Department").Preload("Employee.Position").
		Order("assess_date desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetAssessment returns an assessment with its scores.
func (s *AnalyticsService) GetAssessment(ctx context.Context, id uint) (*model.Assessment, error) {
	var a model.Assessment
	if err := s.db.WithContext(ctx).Preload("Employee").Preload("Scores").First(&a, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &a, nil
}

// CreateAssessment creates an assessment with scores.
func (s *AnalyticsService) CreateAssessment(ctx context.Context, a *model.Assessment, scores []model.AssessmentScore) (*model.Assessment, error) {
	if a.EmployeeID == 0 || a.PlanID == 0 || a.AssessDate.IsZero() {
		return nil, ErrValidation
	}
	if a.Status == "" {
		a.Status = model.AssessmentDone
	}
	if a.AssessmentSource == "" {
		a.AssessmentSource = model.AssessmentSourceManual
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return mapCreateError(err)
		}
		var sum, weight float64
		for i := range scores {
			scores[i].ID = 0
			scores[i].AssessmentID = a.ID
			if err := tx.Create(&scores[i]).Error; err != nil {
				return err
			}
			sum += scores[i].Score * scores[i].Weight
			weight += scores[i].Weight
		}
		if weight > 0 {
			a.CompositeScore = sum / weight
		}
		return tx.Model(a).Update("composite_score", a.CompositeScore).Error
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateAssessment updates assessment text and scores.
func (s *AnalyticsService) UpdateAssessment(ctx context.Context, id uint, a *model.Assessment, scores []model.AssessmentScore) (*model.Assessment, error) {
	var existing model.Assessment
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		a.ID = id
		a.PlanID = existing.PlanID
		a.EmployeeID = existing.EmployeeID
		if a.Status == "" {
			a.Status = existing.Status
		}
		if err := tx.Save(a).Error; err != nil {
			return err
		}
		if err := tx.Where("assessment_id = ?", id).Delete(&model.AssessmentScore{}).Error; err != nil {
			return err
		}
		var sum, weight float64
		for i := range scores {
			scores[i].ID = 0
			scores[i].AssessmentID = id
			if err := tx.Create(&scores[i]).Error; err != nil {
				return err
			}
			sum += scores[i].Score * scores[i].Weight
			weight += scores[i].Weight
		}
		if weight > 0 {
			a.CompositeScore = sum / weight
		}
		return tx.Model(a).Update("composite_score", a.CompositeScore).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetAssessment(ctx, id)
}

// AnalyzeAssessment computes capability analysis for an assessment.
func (s *AnalyticsService) AnalyzeAssessment(ctx context.Context, id uint) (*model.AssessmentAnalysis, error) {
	a, err := s.GetAssessment(ctx, id)
	if err != nil {
		return nil, err
	}
	an := &model.AssessmentAnalysis{CompositeScore: a.CompositeScore}
	for _, sc := range a.Scores {
		met := sc.Score >= sc.TargetLevel
		dim := model.CapabilityDimension{
			Capability: sc.Capability, Score: sc.Score,
			TargetLevel: sc.TargetLevel, Weight: sc.Weight, Met: met,
		}
		an.Dimensions = append(an.Dimensions, dim)
		if met {
			an.MetCapabilities = append(an.MetCapabilities, sc.Capability)
		} else {
			an.UnmetCapabilities = append(an.UnmetCapabilities, sc.Capability)
		}
		if sc.Score < sc.TargetLevel {
			an.WeakCapabilities = append(an.WeakCapabilities, dim)
		}
	}
	return an, nil
}

// --- Reports ---

// ListReports returns training reports.
func (s *AnalyticsService) ListReports(ctx context.Context, opt ListOptions, employeeID uint) ([]model.TrainingReport, int64, error) {
	var items []model.TrainingReport
	query := s.db.WithContext(ctx).Model(&model.TrainingReport{})
	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if len(opt.EmployeeIDs) > 0 {
		query = query.Where("employee_id IN ?", opt.EmployeeIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Employee.Department").Preload("Employee.Position").
		Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetReport returns a report.
func (s *AnalyticsService) GetReport(ctx context.Context, id uint) (*model.TrainingReport, error) {
	var r model.TrainingReport
	if err := s.db.WithContext(ctx).Preload("Employee").First(&r, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &r, nil
}

// CreateReport creates a report.
func (s *AnalyticsService) CreateReport(ctx context.Context, r *model.TrainingReport) (*model.TrainingReport, error) {
	if r.EmployeeID == 0 || r.ReportType == "" {
		return nil, ErrValidation
	}
	if err := s.db.WithContext(ctx).Create(r).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return r, nil
}

// DeleteReport removes a report.
func (s *AnalyticsService) DeleteReport(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.TrainingReport{}, id).Error)
}

// --- Role-specific home screens ---

// EmployeeWorkbench builds the Employee home payload.
func (s *AnalyticsService) EmployeeWorkbench(ctx context.Context, employeeID uint) (*model.EmployeeWorkbench, error) {
	var emp model.Employee
	if err := s.db.WithContext(ctx).Preload("Department").Preload("Position").First(&emp, employeeID).Error; err != nil {
		return nil, mapGormError(err)
	}
	w := &model.EmployeeWorkbench{
		EmployeeID: emp.ID, Name: emp.Name, CurrentStage: emp.CurrentStage,
		Progress: emp.Progress, HireDays: int(time.Since(emp.HireDate).Hours() / 24),
	}
	if emp.Department != nil {
		w.Department = emp.Department.Name
	}
	if emp.Position != nil {
		w.Position = emp.Position.Name
	}
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("created_at desc").First(&plan).Error; err == nil {
		w.Progress = plan.Progress
		w.PlanStatus = string(plan.Status)
		var stage model.TrainingStage
		if err := s.db.WithContext(ctx).Where("plan_id = ? AND stage_number = ?", plan.ID, emp.CurrentStage).First(&stage).Error; err == nil {
			w.StageName = stage.Name
		}
	}
	// Open tasks (pending vs completed). A strict due-today window reads as
	// 0/0 whenever nothing is due exactly today, which confuses people;
	// open-task counts are always meaningful.
	var totalTasks, doneTasks int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ?", employeeID).Count(&totalTasks)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskCompleted).Count(&doneTasks)
	w.TasksToday = int(totalTasks - doneTasks)
	w.TasksTodayDone = int(doneTasks)
	// Recent learning records.
	var records []model.LearningRecord
	s.db.WithContext(ctx).Preload("Task").Where("employee_id = ?", employeeID).Order("start_time desc").Limit(5).Find(&records)
	for _, r := range records {
		title := ""
		if r.Task != nil {
			title = r.Task.Title
		}
		w.RecentRecords = append(w.RecentRecords, model.LearningRecordView{
			ID: r.ID, TaskTitle: title, Content: r.Content, StartTime: r.StartTime,
			DurationMin: r.DurationMin, Status: string(r.Status),
		})
	}
	// Latest assessment capabilities.
	var assess model.Assessment
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("assess_date desc").First(&assess).Error; err == nil {
		var scores []model.AssessmentScore
		s.db.WithContext(ctx).Where("assessment_id = ?", assess.ID).Find(&scores)
		for _, sc := range scores {
			met := sc.Score >= sc.TargetLevel
			cv := model.CapabilityView{Capability: sc.Capability, Score: sc.Score, TargetLevel: sc.TargetLevel, Weight: sc.Weight, Met: met}
			w.Capabilities = append(w.Capabilities, cv)
			if !met {
				w.WeakCapabilities = append(w.WeakCapabilities, cv)
			}
		}
	}
	// Recommendations derived from weak capabilities (human-readable now; AIOS later).
	for _, wc := range w.WeakCapabilities {
		w.Recommendations = append(w.Recommendations, model.RecommendationView{
			Type: "course", Title: "Strengthen: " + wc.Capability,
			Reason: "Score " + fmtScore(wc.Score) + " vs target " + fmtScore(wc.TargetLevel),
		})
	}
	return w, nil
}

// ManagedEmployeeIDs returns the IDs of employees a manager may access
// (direct reports plus everyone in departments they manage).
func (s *AnalyticsService) ManagedEmployeeIDs(ctx context.Context, managerUserID uint) ([]uint, error) {
	return ManagedEmployeeIDs(ctx, s.db, managerUserID)
}

// ManagerCanAccessEmployeeID reports whether the manager may access the
// employee (direct report or member of a department they manage).
func (s *AnalyticsService) ManagerCanAccessEmployeeID(ctx context.Context, managerUserID, employeeID uint) bool {
	var e model.Employee
	if err := s.db.WithContext(ctx).First(&e, employeeID).Error; err != nil {
		return false
	}
	return ManagerCanAccessEmployee(ctx, s.db, managerUserID, &e)
}

// ManagerWorkbench builds the Manager home payload for the manager's own team.
func (s *AnalyticsService) ManagerWorkbench(ctx context.Context, managerUserID uint) (*model.ManagerWorkbench, error) {
	deptIDs, err := ManagedDepartmentIDs(ctx, s.db, managerUserID)
	if err != nil {
		return nil, err
	}
	var emps []model.Employee
	ManagerEmployeeScope(s.db.WithContext(ctx).Preload("Department").Preload("Position"), managerUserID, deptIDs).Find(&emps)
	w := &model.ManagerWorkbench{}
	// "New hires" are employees still inside the 90-day onboarding window.
	for _, e := range emps {
		if time.Since(e.HireDate) <= 90*24*time.Hour {
			w.NewHires++
		}
	}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	riskAll := s.risksFor(ctx, emps)
	for _, e := range emps {
		idx := s.indexOfEmployee(riskAll, e.ID)
		riskStatus := "low"
		risk := model.RiskItem{}
		if idx >= 0 {
			risk = riskAll[idx]
			riskStatus = risk.Severity
		}
		row := model.ManagerTeamRow{
			EmployeeID: e.ID, Name: e.Name, CurrentStage: e.CurrentStage, Progress: e.Progress,
			RiskStatus: riskStatus, ConsecutiveMiss: s.consecutiveMiss(ctx, e.ID),
		}
		if e.Position != nil {
			row.Position = e.Position.Name
		}
		row.TaskRate = s.employeeTaskRate(ctx, e.ID)
		row.WeakCapabilities = s.employeeWeakCaps(ctx, e.ID)
		// RiskCount reflects the number of employees needing attention (non-low risk),
		// not the count of individual weak capabilities. This keeps the KPI consistent
		// with the risk column / risk list shown in the UI.
		if riskStatus != "low" {
			w.RiskCount++
			w.Risks = append(w.Risks, risk)
		}
		w.Team = append(w.Team, row)
		w.AvgProgress += e.Progress
	}
	if len(emps) > 0 {
		w.AvgProgress = w.AvgProgress / float64(len(emps))
	}
	// Team today's task completion.
	sub := ManagerEmployeeScope(s.db.Model(&model.Employee{}).Select("id"), managerUserID, deptIDs)
	var todayTotal, todayDone int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id IN (?) AND due_date >= ? AND due_date < ?", sub, start, end).Count(&todayTotal)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id IN (?) AND due_date >= ? AND due_date < ? AND status = ?", sub, start, end, model.TaskCompleted).Count(&todayDone)
	w.TasksTodayTotal = int(todayTotal)
	w.TasksTodayDone = int(todayDone)
	if w.TasksTodayTotal > 0 {
		w.TasksTodayRate = float64(w.TasksTodayDone) / float64(w.TasksTodayTotal) * 100
	}
	// Pending plans (draft) awaiting review by this manager's team.
	pendingPlanSub := ManagerEmployeeScope(s.db.Model(&model.Employee{}).Select("id"), managerUserID, deptIDs)
	var pendingPlans, pendingAssess int64
	s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("employee_id IN (?) AND status = ?", pendingPlanSub, model.PlanPending).Count(&pendingPlans)
	// Pending assessments for team (none not-yet for stage, count assessments in draft/pending).
	assessSub := ManagerEmployeeScope(s.db.Model(&model.Employee{}).Select("id"), managerUserID, deptIDs)
	s.db.WithContext(ctx).Model(&model.Assessment{}).Where("employee_id IN (?) AND status IN ?", assessSub, []string{string(model.AssessmentDraft), string(model.AssessmentPending)}).Count(&pendingAssess)
	w.PendingPlans = int(pendingPlans)
	w.PendingAssessments = int(pendingAssess)
	// To-dos.
	w.ToDos = s.managerToDos(ctx, w, managerUserID, len(emps))
	return w, nil
}

// AdminCockpit builds the HR/Admin home payload.
func (s *AnalyticsService) AdminCockpit(ctx context.Context) (*model.AdminCockpit, error) {
	c := &model.AdminCockpit{}
	// Stage completion (per stage 1..3).
	stageNames := map[int]string{1: "Onboarding & Fundamentals", 2: "Capability Building", 3: "Independent Competence"}
	for n := 1; n <= 3; n++ {
		var total, completed int64
		row := model.StageCompletionRow{StageNumber: n, Name: stageNames[n]}
		s.db.WithContext(ctx).Model(&model.TrainingStage{}).Where("stage_number = ?", n).Count(&total)
		s.db.WithContext(ctx).Model(&model.TrainingStage{}).Where("stage_number = ? AND status = ?", n, model.StageCompleted).Count(&completed)
		row.Total = int(total)
		row.Completed = int(completed)
		if row.Total > 0 {
			row.Rate = float64(row.Completed) / float64(row.Total) * 100
		}
		c.StageCompletion = append(c.StageCompletion, row)
	}
	// Plan completion rate.
	var plansTotal, plansDone int64
	s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Count(&plansTotal)
	s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("status = ?", model.PlanCompleted).Count(&plansDone)
	if plansTotal > 0 {
		c.PlanCompletionRate = float64(plansDone) / float64(plansTotal) * 100
	}
	// Assessment pass rate (goals_met).
	var assessTotal, assessPass int64
	s.db.WithContext(ctx).Model(&model.Assessment{}).Where("status = ?", model.AssessmentDone).Count(&assessTotal)
	s.db.WithContext(ctx).Model(&model.Assessment{}).Where("status = ? AND goals_met = ?", model.AssessmentDone, true).Count(&assessPass)
	if assessTotal > 0 {
		c.AssessmentPassRate = float64(assessPass) / float64(assessTotal) * 100
	}
	// Department ranking.
	ranking, err := s.DepartmentTrainingRows(ctx)
	if err == nil {
		c.DepartmentRanking = ranking
	}
	// Risk stats.
	allRisks, _ := s.Risks(ctx)
	for _, r := range allRisks {
		switch r.RiskType {
		case "no_plan":
			c.RiskStats.NoPlan++
		case "overdue", "consecutive_overdue":
			c.RiskStats.Overdue++
		case "lagging":
			c.RiskStats.Lagging++
		}
		if r.Severity == "high" {
			c.RiskStats.High++
		}
	}
	// Resource usage.
	var cCourses, cMaterials, cSops int64
	s.db.WithContext(ctx).Model(&model.Course{}).Count(&cCourses)
	s.db.WithContext(ctx).Model(&model.TrainingMaterial{}).Count(&cMaterials)
	s.db.WithContext(ctx).Model(&model.SopDocument{}).Count(&cSops)
	c.ResourceUsage = model.ResourceUsage{Courses: int(cCourses), Materials: int(cMaterials), Sops: int(cSops)}
	return c, nil
}

func (s *AnalyticsService) indexOfEmployee(risks []model.RiskItem, id uint) int {
	for i, r := range risks {
		if r.EmployeeID == id {
			return i
		}
	}
	return -1
}

func (s *AnalyticsService) employeeTaskRate(ctx context.Context, employeeID uint) float64 {
	var total, done int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ?", employeeID).Count(&total)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskCompleted).Count(&done)
	if total == 0 {
		return 0
	}
	return float64(done) / float64(total) * 100
}

func (s *AnalyticsService) consecutiveMiss(ctx context.Context, employeeID uint) int {
	count := 0
	var tasks []model.LearningTask
	s.db.WithContext(ctx).Where("employee_id = ? AND status = ?", employeeID, model.TaskOverdue).Order("due_date desc").Limit(5).Find(&tasks)
	for _, t := range tasks {
		if t.Status == model.TaskOverdue {
			count++
		}
	}
	return count
}

func (s *AnalyticsService) employeeWeakCaps(ctx context.Context, employeeID uint) []string {
	var assess model.Assessment
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("assess_date desc").First(&assess).Error; err != nil {
		return nil
	}
	var scores []model.AssessmentScore
	s.db.WithContext(ctx).Where("assessment_id = ?", assess.ID).Find(&scores)
	var weak []string
	for _, sc := range scores {
		if sc.Score < sc.TargetLevel {
			weak = append(weak, sc.Capability)
		}
	}
	return weak
}

func (s *AnalyticsService) risksFor(ctx context.Context, emps []model.Employee) []model.RiskItem {
	var risks []model.RiskItem
	for _, e := range emps {
		base := model.RiskItem{EmployeeID: e.ID, Employee: e.Name}
		if e.Department != nil {
			base.Department = e.Department.Name
		}
		if e.Position != nil {
			base.Position = e.Position.Name
		}
		if e.TrainingStatus == model.StatusNotStarted {
			base.RiskType = "no_plan"
			base.Severity = "high"
			base.Message = "No training plan has been created yet"
			risks = append(risks, base)
			continue
		}
		if e.TrainingStatus == model.StatusPaused {
			base.RiskType = "paused"
			base.Severity = "high"
			base.Message = "Training plan is paused"
			risks = append(risks, base)
			continue
		}
		var overdue int64
		s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", e.ID, model.TaskOverdue).Count(&overdue)
		if overdue >= 3 {
			base.RiskType = "consecutive_overdue"
			base.Severity = "high"
			base.Message = "3+ overdue tasks in a row"
			risks = append(risks, base)
		} else if overdue > 0 {
			base.RiskType = "overdue"
			base.Severity = "medium"
			base.Message = "Has overdue tasks"
			risks = append(risks, base)
		}
		if e.Progress < 25 && e.CurrentStage >= 2 {
			base.RiskType = "lagging"
			base.Severity = "medium"
			base.Message = "Learning progress is significantly behind"
			risks = append(risks, base)
		}
	}
	return risks
}

func (s *AnalyticsService) managerToDos(ctx context.Context, w *model.ManagerWorkbench, managerUserID uint, newHires int) []model.ToDoItem {
	var todos []model.ToDoItem
	if w.PendingPlans > 0 {
		todos = append(todos, model.ToDoItem{Type: "plan", Title: "Review pending training plans", Detail: fmt.Sprintf("%d plan(s) awaiting your approval", w.PendingPlans)})
	}
	if w.PendingAssessments > 0 {
		todos = append(todos, model.ToDoItem{Type: "assessment", Title: "Complete pending stage assessments", Detail: fmt.Sprintf("%d assessment(s) pending", w.PendingAssessments)})
	}
	if w.RiskCount > 0 {
		todos = append(todos, model.ToDoItem{Type: "risk", Title: "Follow up on at-risk employees", Detail: fmt.Sprintf("%d risk item(s) need attention", w.RiskCount)})
	}
	if newHires == 0 {
		todos = append(todos, model.ToDoItem{Type: "team", Title: "No new hires yet", Detail: "Assign employees to yourself to start coaching"})
	}
	return todos
}

func fmtScore(v float64) string {
	return strconv.FormatFloat(v, 'f', 0, 64)
}

// MyGrowth returns the aggregate personal-growth payload for an employee:
// latest assessment, capability radar scores, composite-score trend, and
// stage summaries. It supports the employee's self-service growth dashboard.
func (s *AnalyticsService) MyGrowth(ctx context.Context, employeeID uint) (*model.AssessmentSnapshot, error) {
	snap := &model.AssessmentSnapshot{EmployeeID: employeeID}

	// All assessments for this employee, newest first.
	var all []model.Assessment
	if err := s.db.WithContext(ctx).Preload("Scores").
		Where("employee_id = ?", employeeID).
		Order("assess_date desc").Find(&all).Error; err != nil {
		return nil, mapGormError(err)
	}

	if len(all) > 0 {
		latest := all[0]
		snap.Latest = &latest
		snap.Scores = latest.Scores

		var caps []model.CapabilitySummary
		for _, sc := range latest.Scores {
			caps = append(caps, model.CapabilitySummary{
				Capability: sc.Capability, Score: sc.Score,
				TargetLevel: sc.TargetLevel, Met: sc.Score >= sc.TargetLevel,
			})
		}
		snap.Capabilities = caps
	}

	// Composite-score trend (oldest -> newest) and stage summaries.
	for i := len(all) - 1; i >= 0; i-- {
		a := all[i]
		snap.Trend = append(snap.Trend, model.AssessmentTrendPoint{
			StageNumber: a.StageNumber, AssessDate: a.AssessDate.Format("2006-01-02"),
			CompositeScore: a.CompositeScore, GoalsMet: a.GoalsMet,
		})
		snap.Stages = append(snap.Stages, model.AssessmentStageRow{
			StageNumber: a.StageNumber, CompositeScore: a.CompositeScore, Status: string(a.Status),
		})
	}

	if len(all) == 0 {
		// No assessments yet: derive the snapshot from real learning data
		// (graded quizzes and stage progress) so the dashboard still shows
		// the employee's genuine growth. Nothing is fabricated.
		s.fillGrowthFromLearning(ctx, snap, employeeID)
	}

	return snap, nil
}

// fillGrowthFromLearning builds the growth snapshot from graded test tasks
// (capability radar + score trend) and plan-stage progress (stage summary)
// for employees who have learning data but no assessments yet.
func (s *AnalyticsService) fillGrowthFromLearning(ctx context.Context, snap *model.AssessmentSnapshot, employeeID uint) {
	var tests []model.LearningTask
	s.db.WithContext(ctx).Where("employee_id = ? AND task_type = ? AND status = ? AND score IS NOT NULL",
		employeeID, model.TaskTest, model.TaskCompleted).
		Order("completed_at asc").Find(&tests)

	// Map stage IDs to stage numbers for the trend points.
	stageNum := map[uint]int{}
	var stageIDs []uint
	for _, t := range tests {
		stageIDs = append(stageIDs, t.StageID)
	}
	if len(stageIDs) > 0 {
		var stages []model.TrainingStage
		s.db.WithContext(ctx).Where("id IN ?", stageIDs).Find(&stages)
		for _, st := range stages {
			stageNum[st.ID] = st.StageNumber
		}
	}

	for _, t := range tests {
		if t.Score == nil {
			continue
		}
		score := *t.Score
		snap.Capabilities = append(snap.Capabilities, model.CapabilitySummary{
			Capability: t.Title, Score: score, TargetLevel: 70, Met: score >= 70,
		})
		date := ""
		if t.CompletedAt != nil {
			date = t.CompletedAt.Format("2006-01-02")
		} else if !t.UpdatedAt.IsZero() {
			// Seeded completions may lack completed_at; fall back to the
			// record's real update time instead of showing a blank date.
			date = t.UpdatedAt.Format("2006-01-02")
		}
		snap.Trend = append(snap.Trend, model.AssessmentTrendPoint{
			StageNumber: stageNum[t.StageID], AssessDate: date,
			CompositeScore: score, GoalsMet: score >= 70,
		})
	}

	// Stage summary from the employee's latest plan (real stage progress).
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).
		Order("created_at desc").First(&plan).Error; err == nil {
		// "Task completion" radar dimension: the real plan completion rate.
		snap.Capabilities = append(snap.Capabilities, model.CapabilitySummary{
			Capability: "Task completion", Score: plan.Progress, TargetLevel: 80, Met: plan.Progress >= 80,
		})
		var stages []model.TrainingStage
		s.db.WithContext(ctx).Where("plan_id = ?", plan.ID).Order("stage_number").Find(&stages)
		for _, st := range stages {
			snap.Stages = append(snap.Stages, model.AssessmentStageRow{
				StageNumber: st.StageNumber, CompositeScore: st.Progress, Status: string(st.Status),
			})
		}
	}

	// "Learning record avg" radar dimension: the employee's real average
	// score across all learning records that carry a score.
	var recCount int64
	s.db.WithContext(ctx).Model(&model.LearningRecord{}).
		Where("employee_id = ? AND test_score IS NOT NULL", employeeID).Count(&recCount)
	if recCount > 0 {
		var avg float64
		s.db.WithContext(ctx).Model(&model.LearningRecord{}).
			Where("employee_id = ? AND test_score IS NOT NULL", employeeID).
			Select("COALESCE(AVG(test_score),0)").Scan(&avg)
		snap.Capabilities = append(snap.Capabilities, model.CapabilitySummary{
			Capability: "Learning record avg", Score: avg, TargetLevel: 70, Met: avg >= 70,
		})
	}
}
