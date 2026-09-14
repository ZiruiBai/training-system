package service

import (
	"context"
	"strings"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// TemplateService handles training templates and generating plan stages/tasks.
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService creates a TemplateService.
func NewTemplateService(db *gorm.DB) *TemplateService { return &TemplateService{db: db} }

// ListTemplates returns reusable training templates.
func (s *TemplateService) ListTemplates(ctx context.Context, opt ListOptions, positionID uint, status string) ([]model.TrainingTemplate, int64, error) {
	var items []model.TrainingTemplate
	query := s.db.WithContext(ctx).Model(&model.TrainingTemplate{})
	if positionID > 0 {
		query = query.Where("position_id = ?", positionID)
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
	if err := query.Preload("Position").Preload("Stages").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetTemplate returns one template with its stages.
func (s *TemplateService) GetTemplate(ctx context.Context, id uint) (*model.TrainingTemplate, error) {
	var t model.TrainingTemplate
	if err := s.db.WithContext(ctx).Preload("Position").Preload("Stages").First(&t, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	return &t, nil
}

// CreateTemplate stores a template with its stages.
func (s *TemplateService) CreateTemplate(ctx context.Context, t *model.TrainingTemplate, stages []model.TrainingTemplateStage) (*model.TrainingTemplate, error) {
	if t.Title == "" {
		return nil, ErrValidation
	}
	if t.CycleDays <= 0 {
		t.CycleDays = 90
	}
	if t.Status == "" {
		t.Status = model.TemplateDraft
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return mapCreateError(err)
		}
		// Default 3 stages if none supplied.
		if len(stages) == 0 {
			stages = defaultTemplateStages()
		}
		for i := range stages {
			stages[i].ID = 0
			stages[i].TemplateID = t.ID
			if stages[i].Name == "" {
				return ErrValidation
			}
			if err := tx.Create(&stages[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetTemplate(ctx, t.ID)
}

// UpdateTemplate replaces a template's metadata and stages.
func (s *TemplateService) UpdateTemplate(ctx context.Context, id uint, t *model.TrainingTemplate, stages []model.TrainingTemplateStage) (*model.TrainingTemplate, error) {
	var existing model.TrainingTemplate
	if err := s.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return nil, mapGormError(err)
	}
	if t.Title == "" {
		return nil, ErrValidation
	}
	t.ID = id
	t.CreatedAt = existing.CreatedAt
	if t.CycleDays <= 0 {
		t.CycleDays = existing.CycleDays
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(t).Error; err != nil {
			return mapCreateError(err)
		}
		if err := tx.Where("template_id = ?", id).Delete(&model.TrainingTemplateStage{}).Error; err != nil {
			return err
		}
		if len(stages) == 0 {
			stages = defaultTemplateStages()
		}
		for i := range stages {
			stages[i].ID = 0
			stages[i].TemplateID = id
			if stages[i].Name == "" {
				return ErrValidation
			}
			if err := tx.Create(&stages[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetTemplate(ctx, id)
}

// DeleteTemplate removes a template and its stages.
func (s *TemplateService) DeleteTemplate(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&model.TrainingTemplateStage{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.TrainingTemplate{}, id).Error; err != nil {
			return mapGormError(err)
		}
		return nil
	})
}

// TemplateStagesAndTasks is the generated content for a plan.
type TemplateStagesAndTasks struct {
	Stages []model.TrainingStage
	Tasks  []model.LearningTask
}

// GeneratePlanContent produces plan stages and seed tasks from a template.
func (s *TemplateService) GeneratePlanContent(ctx context.Context, templateID uint, plan *model.TrainingPlan) (*TemplateStagesAndTasks, error) {
	tmpl, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	out := &TemplateStagesAndTasks{}
	start := plan.StartDate
	for i, ts := range tmpl.Stages {
		stageStart := start.AddDate(0, 0, i*30)
		stageEnd := start.AddDate(0, 0, i*30+29)
		stage := model.TrainingStage{
			PlanID: plan.ID, StageNumber: ts.StageNumber, Name: ts.Name,
			Objectives: ts.Objectives, StartDate: stageStart, EndDate: stageEnd,
			Status: model.StageNotStarted, Progress: 0,
		}
		out.Stages = append(out.Stages, stage)
		// Seed tasks from sample task titles (comma/semicolon separated).
		for j, title := range splitTitles(ts.SampleTaskTitles) {
			if j >= 8 { // cap tasks per stage for seed
				break
			}
			out.Tasks = append(out.Tasks, model.LearningTask{
				EmployeeID:    plan.EmployeeID,
				PlanID:        plan.ID,
				Title:         title,
				TaskType:      model.TaskLearning,
				StartDate:     stageStart,
				DueDate:       stageEnd,
				Status:        model.TaskPending,
				Priority:      model.PriorityMedium,
				CreatedSource: model.SourceManual,
			})
		}
	}
	if len(out.Stages) == 0 {
		out.Stages = defaultStages(plan)
	}
	return out, nil
}

// CreatePlanFromTemplate creates a training plan (with its 3 stages) and seeds
// learning tasks from the template in a single transaction.
func (s *TemplateService) CreatePlanFromTemplate(ctx context.Context, plan *model.TrainingPlan, templateID uint, createdBy *uint) (*model.TrainingPlan, error) {
	content, err := s.GeneratePlanContent(ctx, templateID, plan)
	if err != nil {
		return nil, err
	}
	plan.Status = model.PlanDraft
	plan.CurrentStage = 1
	plan.Progress = 0
	plan.CreatedBy = createdBy
	plan.CreatedSource = model.SourceManual
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(plan).Error; err != nil {
			return mapCreateError(err)
		}
		stageIDs := make(map[int]uint)
		for i := range content.Stages {
			content.Stages[i].PlanID = plan.ID
			if err := tx.Create(&content.Stages[i]).Error; err != nil {
				return err
			}
			stageIDs[content.Stages[i].StageNumber] = content.Stages[i].ID
		}
		for i := range content.Tasks {
			content.Tasks[i].PlanID = plan.ID
			content.Tasks[i].StageID = stageIDs[defaultStageNumberFor(i)]
			if err := tx.Create(&content.Tasks[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Preload("Employee.Department").Preload("Employee.Position").First(plan, plan.ID).Error; err != nil {
		return nil, err
	}
	return plan, nil
}

// defaultStageNumberFor maps a seed task index to a stage number (0-29 -> 1, etc).
func defaultStageNumberFor(i int) int {
	idx := i / 8
	if idx > 2 {
		idx = 2
	}
	return idx + 1
}

func defaultTemplateStages() []model.TrainingTemplateStage {
	return []model.TrainingTemplateStage{
		{StageNumber: 1, Name: "Onboarding & Fundamentals",
			Objectives:       "Get familiar with the company, department, role duties, policies and basic tools; complete induction training.",
			SampleTaskTitles: "Read employee handbook;Complete department onboarding;Watch AI fundamentals course;Take induction quiz",
			DurationDays:     30},
		{StageNumber: 2, Name: "Capability Building",
			Objectives:       "Master core job knowledge and SOPs, complete real tasks and begin working independently to build core competence.",
			SampleTaskTitles: "Learn position SOPs;Complete hands-on task;Shadow a senior colleague;Write weekly reflection",
			DurationDays:     30},
		{StageNumber: 3, Name: "Independent Competence",
			Objectives:       "Complete work independently, solve problems, meet the capability standard, pass the final assessment.",
			SampleTaskTitles: "Deliver an independent task;Solve a real problem;Final capability review;Final assessment",
			DurationDays:     30},
	}
}

// splitTitles splits a semicolon/comma separated string into trimmed titles.
func splitTitles(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{"Complete a stage learning task"}
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = append(out, "Complete a stage learning task")
	}
	return out
}
