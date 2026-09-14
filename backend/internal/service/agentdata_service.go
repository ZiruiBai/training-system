package service

import (
	"context"
	"encoding/json"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// AgentDataService exposes read-only business data for the G.AIOS agent tools.
type AgentDataService struct {
	db *gorm.DB
}

// NewAgentDataService creates an AgentDataService.
func NewAgentDataService(db *gorm.DB) *AgentDataService { return &AgentDataService{db: db} }

// EmployeeProfile queries the employee profile by id.
func (s *AgentDataService) EmployeeProfile(ctx context.Context, employeeID uint) (*model.AgentEmployeeProfile, error) {
	var emp model.Employee
	if err := s.db.WithContext(ctx).Preload("Department").Preload("Position").First(&emp, employeeID).Error; err != nil {
		return nil, mapGormError(err)
	}
	p := &model.AgentEmployeeProfile{
		EmployeeID: emp.ID, Name: emp.Name, CurrentStage: emp.CurrentStage,
		Status: string(emp.TrainingStatus), JoinDate: emp.HireDate.Format("2006-01-02"),
	}
	if emp.Department != nil {
		p.Department = emp.Department.Name
	}
	if emp.Position != nil {
		p.Position = emp.Position.Name
	}
	return p, nil
}

// EmployeeLearningData aggregates the learning situation for an employee.
func (s *AgentDataService) EmployeeLearningData(ctx context.Context, employeeID uint) (*model.AgentLearningData, error) {
	var emp model.Employee
	if err := s.db.WithContext(ctx).First(&emp, employeeID).Error; err != nil {
		return nil, mapGormError(err)
	}
	d := &model.AgentLearningData{EmployeeID: emp.ID, CurrentStage: emp.CurrentStage}

	// Active (latest) plan.
	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("created_at desc").First(&plan).Error; err == nil {
		d.PlanID = plan.ID
		d.PlanStatus = string(plan.Status)
		d.PlanProgress = plan.Progress
	} else {
		d.PlanProgress = emp.Progress
	}

	// Task counts.
	var total, done, overdue int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ?", employeeID).Count(&total)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskCompleted).Count(&done)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND status = ?", employeeID, model.TaskOverdue).Count(&overdue)
	d.TotalTasks = int(total)
	d.CompletedTasks = int(done)
	d.IncompleteTasks = int(total - done)
	d.OverdueTasks = int(overdue)

	// Course completion: distinct courses from tasks.
	var courses, coursesDone int64
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND course_id IS NOT NULL", employeeID).Distinct("course_id").Count(&courses)
	s.db.WithContext(ctx).Model(&model.LearningTask{}).Where("employee_id = ? AND course_id IS NOT NULL AND status = ?", employeeID, model.TaskCompleted).Distinct("course_id").Count(&coursesDone)
	d.TotalCourses = int(courses)
	d.CompletedCourses = int(coursesDone)
	if courses > 0 {
		d.CourseRate = float64(coursesDone) / float64(courses) * 100
	}

	// Test scores from learning records.
	var scores []model.LearningRecord
	s.db.WithContext(ctx).Where("employee_id = ? AND test_score IS NOT NULL", employeeID).Order("start_time asc").Find(&scores)
	var sum float64
	for _, sc := range scores {
		if sc.TestScore != nil {
			d.TestScores = append(d.TestScores, *sc.TestScore)
			sum += *sc.TestScore
		}
	}
	if len(d.TestScores) > 0 {
		d.AvgTestScore = sum / float64(len(d.TestScores))
	}

	// Capabilities from the latest assessment.
	var assess model.Assessment
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("assess_date desc").First(&assess).Error; err == nil {
		var capScores []model.AssessmentScore
		s.db.WithContext(ctx).Where("assessment_id = ?", assess.ID).Find(&capScores)
		for _, cs := range capScores {
			d.Capabilities = append(d.Capabilities, model.AgentCapabilityScore{
				Capability: cs.Capability, Score: cs.Score, TargetLevel: cs.TargetLevel,
				Met: cs.Score >= cs.TargetLevel,
			})
		}
	}

	// All past assessments summary.
	var allAssess []model.Assessment
	s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Order("assess_date desc").Find(&allAssess)
	for _, a := range allAssess {
		d.Assessments = append(d.Assessments, model.AgentAssessmentSummary{
			StageNumber: a.StageNumber, AssessDate: a.AssessDate.Format("2006-01-02"),
			CompositeScore: a.CompositeScore, GoalsMet: a.GoalsMet, Status: string(a.Status),
		})
	}
	return d, nil
}

// EmployeeTasks returns recent learning tasks for an employee.
func (s *AgentDataService) EmployeeTasks(ctx context.Context, employeeID uint, limit int) ([]model.AgentTask, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	var tasks []model.LearningTask
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).
		Order("due_date desc").Limit(limit).Find(&tasks).Error; err != nil {
		return nil, err
	}
	out := make([]model.AgentTask, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, model.AgentTask{
			TaskID: t.ID, Title: t.Title, Description: t.Description, TaskType: string(t.TaskType),
			DueDate: t.DueDate.Format("2006-01-02"), Status: string(t.Status),
			Score: t.Score,
		})
	}
	return out, nil
}

// EmployeeContext assembles the employee's business data into a single JSON
// blob so the backend can inject it into the agent request. This keeps the
// agent answering only from backend-provided data and never querying the DB.
func (s *AgentDataService) EmployeeContext(ctx context.Context, employeeID uint) (string, error) {
	profile, err := s.EmployeeProfile(ctx, employeeID)
	if err != nil {
		return "", err
	}
	learning, err := s.EmployeeLearningData(ctx, employeeID)
	if err != nil {
		return "", err
	}
	tasks, err := s.EmployeeTasks(ctx, employeeID, 15)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"profile":      profile,
		"learning":     learning,
		"recent_tasks": tasks,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
