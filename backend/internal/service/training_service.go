package service

import (
	"context"
	"time"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// TrainingService handles plans, stages, tasks and learning records.
type TrainingService struct {
	db *gorm.DB
}

// NewTrainingService creates a TrainingService.
func NewTrainingService(db *gorm.DB) *TrainingService { return &TrainingService{db: db} }

// --- Plans ---

// ListPlans returns training plans with filters.
func (s *TrainingService) ListPlans(ctx context.Context, opt ListOptions, employeeID, departmentID uint, status string) ([]model.TrainingPlan, int64, error) {
	var items []model.TrainingPlan
	query := s.db.WithContext(ctx).Model(&model.TrainingPlan{})
	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if departmentID > 0 {
		query = query.Where("employee_id IN (?)", s.db.Model(&model.Employee{}).Select("id").Where("department_id = ?", departmentID))
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if len(opt.EmployeeIDs) > 0 {
		query = query.Where("employee_id IN ?", opt.EmployeeIDs)
	}
	if opt.Search != "" {
		like := "%" + opt.Search + "%"
		query = query.Where("id IN (?)", s.db.Model(&model.Employee{}).Select("id").Where("name LIKE ?", like))
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

// GetPlan returns a plan with stages.
func (s *TrainingService) GetPlan(ctx context.Context, id uint) (*model.TrainingPlan, error) {
	var p model.TrainingPlan
	if err := s.db.WithContext(ctx).Preload("Employee.Department").Preload("Employee.Position").
		Preload("TrainingStages").First(&p, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &p, nil
}

// CreatePlanWithStages creates a plan and its 3 standard stages in one transaction.
func (s *TrainingService) CreatePlanWithStages(ctx context.Context, plan *model.TrainingPlan, createdBy *uint, source model.CreatedSource) (*model.TrainingPlan, error) {
	if plan.EmployeeID == 0 || plan.StartDate.IsZero() || plan.EndDate.IsZero() {
		return nil, ErrValidation
	}
	if plan.CycleDays <= 0 {
		plan.CycleDays = 90
	}
	plan.Status = model.PlanDraft
	plan.CurrentStage = 1
	plan.Progress = 0
	plan.CreatedBy = createdBy
	plan.CreatedSource = source

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return mapCreateError(err)
		}
		// One current plan per employee: mark any active plan as published/in_progress.
		// Standard 3 stages.
		stages := defaultStages(plan)
		for i := range stages {
			if err := tx.Create(&stages[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// UpdatePlan updates plan metadata.
func (s *TrainingService) UpdatePlan(ctx context.Context, id uint, plan *model.TrainingPlan) (*model.TrainingPlan, error) {
	var existing model.TrainingPlan
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if plan.EndDate.Before(plan.StartDate) {
		return nil, ErrValidation
	}
	plan.ID = id
	plan.EmployeeID = existing.EmployeeID
	plan.CreatedAt = existing.CreatedAt
	if plan.CreatedSource == "" {
		plan.CreatedSource = existing.CreatedSource
	}
	if err := s.db.WithContext(ctx).Save(plan).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return plan, nil
}

// Transition changes a plan's lifecycle status.
func (s *TrainingService) Transition(ctx context.Context, id uint, newStatus model.PlanStatus, actor *model.Principal) (*model.TrainingPlan, error) {
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).First(&plan, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if err := validatePlanTransition(plan.Status, newStatus); err != nil {
		return nil, err
	}
	updates := map[string]any{"status": newStatus}
	if newStatus == model.PlanPublished || newStatus == model.PlanInProgress {
		// Publishing marks the first stage in progress.
		updates["status"] = model.PlanInProgress
		var emp model.Employee
		s.db.WithContext(ctx).First(&emp, plan.EmployeeID)
		updates["current_stage"] = 1
		tx := s.db.WithContext(ctx).Begin()
		if err := tx.Model(&plan).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if errors := tx.Model(&model.TrainingStage{}).Where("plan_id = ? AND stage_number = ?", id, 1).Update("status", model.StageInProgress).Error; errors != nil {
			tx.Rollback()
			return nil, errors
		}
		if err := tx.Model(&model.Employee{}).Where("id = ?", emp.ID).Update("training_status", model.StatusInProgress).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
	} else if newStatus == model.PlanPending {
		plan.ReviewedBy = &actor.UserID
		t := time.Now().UTC()
		plan.ReviewedAt = &t
		if err := s.db.WithContext(ctx).Save(&plan).Error; err != nil {
			return nil, err
		}
		updates = map[string]any{"status": newStatus, "reviewed_by": actor.UserID, "reviewed_at": t}
		if err := s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Model(&plan).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetPlan(ctx, id)
}

func validatePlanTransition(from, to model.PlanStatus) error {
	allowed := map[model.PlanStatus][]model.PlanStatus{
		model.PlanDraft:      {model.PlanPending, model.PlanPaused},
		model.PlanPending:    {model.PlanDraft, model.PlanInProgress},
		model.PlanInProgress: {model.PlanCompleted, model.PlanPaused},
		model.PlanPublished:  {model.PlanInProgress, model.PlanPaused},
		model.PlanPaused:     {model.PlanInProgress},
		model.PlanCompleted:  {},
	}
	for _, allow := range allowed[from] {
		if allow == to {
			return nil
		}
	}
	return ErrConflict
}

func defaultStages(plan *model.TrainingPlan) []model.TrainingStage {
	sd := plan.StartDate
	return []model.TrainingStage{
		{PlanID: plan.ID, StageNumber: 1, Name: "Onboarding & Fundamentals",
			Objectives: "Get familiar with the company, department, role duties, policies and basic tools; complete induction training.",
			StartDate:  sd, EndDate: sd.AddDate(0, 0, 29), Status: model.StageNotStarted, Progress: 0},
		{PlanID: plan.ID, StageNumber: 2, Name: "Capability Building",
			Objectives: "Master core job knowledge and SOPs, complete real tasks and begin working independently to build core competence.",
			StartDate:  sd.AddDate(0, 0, 30), EndDate: sd.AddDate(0, 0, 59), Status: model.StageNotStarted, Progress: 0},
		{PlanID: plan.ID, StageNumber: 3, Name: "Independent Competence",
			Objectives: "Complete work independently, solve problems, meet the capability standard, pass the final assessment.",
			StartDate:  sd.AddDate(0, 0, 60), EndDate: sd.AddDate(0, 0, 89), Status: model.StageNotStarted, Progress: 0},
	}
}

// --- Stages ---

// ListStages returns stages for a plan.
func (s *TrainingService) ListStages(ctx context.Context, planID uint) ([]model.TrainingStage, error) {
	var items []model.TrainingStage
	if err := s.db.WithContext(ctx).Where("plan_id = ?", planID).Order("stage_number asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// --- Tasks ---

// ListTasks returns tasks with filters.
func (s *TrainingService) ListTasks(ctx context.Context, opt ListOptions, employeeID, planID, stageID uint, status string) ([]model.LearningTask, int64, error) {
	var items []model.LearningTask
	query := s.db.WithContext(ctx).Model(&model.LearningTask{})
	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if planID > 0 {
		query = query.Where("plan_id = ?", planID)
	}
	if stageID > 0 {
		query = query.Where("stage_id = ?", stageID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if opt.Search != "" {
		query = query.Where("title LIKE ?", "%"+opt.Search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Stage").Order("due_date asc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetTask returns a task.
func (s *TrainingService) GetTask(ctx context.Context, id uint) (*model.LearningTask, error) {
	var t model.LearningTask
	if err := s.db.WithContext(ctx).Preload("Stage").First(&t, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &t, nil
}

// CreateTask creates a task.
func (s *TrainingService) CreateTask(ctx context.Context, task *model.LearningTask) (*model.LearningTask, error) {
	if task.Title == "" || task.EmployeeID == 0 || task.PlanID == 0 {
		return nil, ErrValidation
	}
	if task.Status == "" {
		task.Status = model.TaskPending
	}
	if task.Priority == "" {
		task.Priority = model.PriorityMedium
	}
	if task.TaskType == "" {
		task.TaskType = model.TaskLearning
	}
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return task, nil
}

// UpdateTask updates task content.
func (s *TrainingService) UpdateTask(ctx context.Context, id uint, task *model.LearningTask) (*model.LearningTask, error) {
	var existing model.LearningTask
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if task.Title == "" {
		return nil, ErrValidation
	}
	task.ID = id
	task.EmployeeID = existing.EmployeeID
	task.PlanID = existing.PlanID
	task.StageID = existing.StageID
	task.CreatedAt = existing.CreatedAt
	task.CreatedSource = existing.CreatedSource
	if task.Status == "" {
		task.Status = existing.Status
	}
	if err := s.db.WithContext(ctx).Save(task).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return task, nil
}

// CompleteTask marks a task complete and records its outcome and score.
func (s *TrainingService) CompleteTask(ctx context.Context, id uint, outcome string, score *float64, answers []int) (*model.LearningTask, error) {
	var task model.LearningTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	now := time.Now().UTC()

	// Auto-grade when answers are supplied for a quiz task.
	if len(answers) > 0 && len(task.Quiz) > 0 {
		score = gradeQuiz(task.Quiz, answers)
	}

	updates := map[string]any{
		"status": model.TaskCompleted, "completed_at": now, "outcome": outcome,
	}
	if score != nil {
		updates["score"] = *score
	}
	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	s.upsertLearningRecordForTask(ctx, task, outcome, score, now)
	s.recomputeStageProgress(ctx, task.StageID)
	s.recomputePlanProgress(ctx, task.PlanID, task.EmployeeID)
	return s.GetTask(ctx, id)
}

// gradeQuiz scores a set of answers against a quiz, returning a 0-100 score.
// Answers[i] is the chosen option (arbitrary); a correct match counts.
func gradeQuiz(quiz []model.QuizQuestion, answers []int) *float64 {
	if len(quiz) == 0 {
		return nil
	}
	var correct int
	for i := range quiz {
		if i < len(answers) && answers[i] == quiz[i].Answer {
			correct++
		}
	}
	v := float64(correct) / float64(len(quiz)) * 100
	return &v
}

// upsertLearningRecordForTask creates or updates the learning record for a
// completed task so that the employee's test score and outcome are persisted
// and shown in Learning Records/My Workbench instead of being derived only
// from seed data.
func (s *TrainingService) upsertLearningRecordForTask(ctx context.Context, task model.LearningTask, outcome string, score *float64, now time.Time) {
	var rec model.LearningRecord
	if err := s.db.WithContext(ctx).Where("task_id = ?", task.ID).First(&rec).Error; err != nil {
		// No record yet -> create one.
		start := task.StartDate
		if start.IsZero() {
			start = now
		}
		rec = model.LearningRecord{
			EmployeeID: task.EmployeeID, TaskID: task.ID,
			Content: task.Title, StartTime: start, EndTime: &now,
			DurationMin: int(task.EstimateHours * 60), Status: model.TaskCompleted,
			Outcome: outcome, TestScore: score,
		}
		_ = s.db.WithContext(ctx).Create(&rec).Error
		return
	}
	// Update the existing record.
	upd := map[string]any{"status": model.TaskCompleted, "end_time": now, "outcome": outcome}
	if score != nil {
		upd["test_score"] = *score
	}
	_ = s.db.WithContext(ctx).Model(&rec).Updates(upd).Error
}

// DeleteTask removes a task.
func (s *TrainingService) DeleteTask(ctx context.Context, id uint) error {
	return mapGormError(s.db.WithContext(ctx).Delete(&model.LearningTask{}, id).Error)
}

func (s *TrainingService) recomputeStageProgress(ctx context.Context, stageID uint) {
	var total, done int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("stage_id = ?", stageID).Count(&total)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("stage_id = ? AND status = ?", stageID, model.TaskCompleted).Count(&done)
	var p float64
	if total > 0 {
		p = float64(done) / float64(total) * 100
	}
	status := model.StageCompleted
	if done < total {
		status = model.StageInProgress
		if done == 0 {
			status = model.StageInProgress
		}
	}
	s.db.WithContext(ctx).Model(&model.TrainingStage{}).Where("id = ?", stageID).
		Updates(map[string]any{"progress": p, "status": status})

	// Auto-advance current_stage when this stage is fully completed.
	if total > 0 && done >= total {
		s.advanceCurrentStage(ctx, stageID)
	}
}

// advanceCurrentStage advances the plan's (and employee's) current_stage to the
// next stage when the given stage is fully completed, capped at the plan's
// number of stages. This keeps the employee's "Stage N" indicator moving.
func (s *TrainingService) advanceCurrentStage(ctx context.Context, stageID uint) {
	var stage model.TrainingStage
	if err := s.db.WithContext(ctx).First(&stage, stageID).Error; err != nil {
		return
	}
	var stageCount int64
	s.db.WithContext(ctx).Model(&model.TrainingStage{}).Where("plan_id = ?", stage.PlanID).Count(&stageCount)
	next := stage.StageNumber + 1
	if int64(stage.StageNumber) >= stageCount {
		next = stage.StageNumber // already at last stage; no further advance
	}
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).First(&plan, stage.PlanID).Error; err != nil {
		return
	}
	// Only move forward, never backwards.
	if plan.CurrentStage < next {
		s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("id = ?", plan.ID).Update("current_stage", next)
		s.db.WithContext(ctx).Model(&model.Employee{}).Where("id = ?", plan.EmployeeID).Update("current_stage", next)
	}
}

func (s *TrainingService) recomputePlanProgress(ctx context.Context, planID uint, employeeID uint) {
	var avg float64
	s.db.WithContext(ctx).Model(&model.TrainingStage{}).Where("plan_id = ?", planID).Select("COALESCE(AVG(progress),0)").Scan(&avg)
	var emp model.Employee
	if err := s.db.WithContext(ctx).First(&emp, employeeID).Error; err == nil {
		planStatus := model.PlanInProgress
		if avg >= 99.9 {
			planStatus = model.PlanCompleted
		}
		s.db.WithContext(ctx).Model(&model.TrainingPlan{}).Where("id = ?", planID).
			Updates(map[string]any{"progress": avg, "status": planStatus})
		s.db.WithContext(ctx).Model(&model.Employee{}).Where("id = ?", employeeID).
			Update("progress", avg)
	}
}

// --- Learning records ---

// ListRecords returns learning records.
func (s *TrainingService) ListRecords(ctx context.Context, opt ListOptions, employeeID, taskID uint) ([]model.LearningRecord, int64, error) {
	var items []model.LearningRecord
	query := s.db.WithContext(ctx).Model(&model.LearningRecord{})
	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	if len(opt.EmployeeIDs) > 0 {
		query = query.Where("employee_id IN ?", opt.EmployeeIDs)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(opt.Page, opt.PageSize)
	if err := query.Preload("Task").Preload("Employee").Order("start_time desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreateRecord adds a learning record for a task.
func (s *TrainingService) CreateRecord(ctx context.Context, rec *model.LearningRecord) (*model.LearningRecord, error) {
	if rec.EmployeeID == 0 || rec.TaskID == 0 {
		return nil, ErrValidation
	}
	if rec.Status == "" {
		rec.Status = model.TaskCompleted
	}
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		return nil, mapCreateError(err)
	}
	return rec, nil
}

// LearningStats aggregates learning analytics for an employee.
type LearningStats struct {
	TotalHours   float64 `json:"total_hours"`
	TaskRate     float64 `json:"task_rate"`
	CourseRate   float64 `json:"course_rate"`
	AvgTestScore float64 `json:"avg_test_score"`
	StreakDays   int     `json:"streak_days"`
}

// LearningStats returns analytics for an employee.
func (s *TrainingService) LearningStats(ctx context.Context, employeeID uint) (*LearningStats, error) {
	stats := &LearningStats{}
	var totalMin int64
	s.db.WithContext(ctx).Model(&model.LearningRecord{}).Where("employee_id = ?", employeeID).
		Select("COALESCE(SUM(duration_min),0)").Scan(&totalMin)
	stats.TotalHours = float64(totalMin) / 60.0

	var taskTotal, taskDone int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ?", employeeID).Count(&taskTotal)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskCompleted).Count(&taskDone)
	if taskTotal > 0 {
		stats.TaskRate = float64(taskDone) / float64(taskTotal) * 100
	}

	var avgScore *float64
	s.db.WithContext(ctx).Model(&model.LearningRecord{}).Where("employee_id = ? AND test_score IS NOT NULL", employeeID).
		Select("COALESCE(AVG(test_score),0)").Scan(&avgScore)
	if avgScore != nil {
		stats.AvgTestScore = *avgScore
	}

	// Course completion: distinct courses referenced by completed tasks.
	var courseCount, courseDone int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND course_id IS NOT NULL", employeeID).
		Distinct("course_id").Count(&courseCount)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND course_id IS NOT NULL AND status = ?", employeeID, model.TaskCompleted).
		Distinct("course_id").Count(&courseDone)
	if courseCount > 0 {
		stats.CourseRate = float64(courseDone) / float64(courseCount) * 100
	}
	return stats, nil
}
