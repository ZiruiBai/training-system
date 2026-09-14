package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/training-platform/internal/model"
	"gorm.io/gorm"
)

// CoachingService generates and applies manager-reviewed coaching recommendations.
type CoachingService struct {
	db *gorm.DB
}

// NewCoachingService creates a CoachingService.
func NewCoachingService(db *gorm.DB) *CoachingService {
	return &CoachingService{db: db}
}

// Analysis outcomes that are normal business results, not system failures.
// They map to 400 with their own codes so the UI can show a friendly,
// non-alarming message instead of a generic validation error.
var (
	ErrNoGaps         = errors.New("learning data looks good so far; no significant gaps detected")
	ErrNoLearningData = errors.New("no learning data yet (no assessments or scored test tasks)")
)

// Analyze builds a draft coaching recommendation for an employee from real
// learning data (assessment scores, test task scores). No fabricated data:
// every gap carries its evidence and every item references an existing
// course/material of the employee's position.
func (s *CoachingService) Analyze(ctx context.Context, employeeID, managerID uint) (*model.CoachingRecommendation, error) {
	var emp model.Employee
	if err := s.db.WithContext(ctx).First(&emp, employeeID).Error; err != nil {
		return nil, ErrNotFound
	}
	gaps := s.diagnoseGaps(ctx, &emp)
	if len(gaps) == 0 {
		if s.hasLearningEvidence(ctx, emp.ID) {
			return nil, ErrNoGaps
		}
		return nil, ErrNoLearningData
	}
	rec := &model.CoachingRecommendation{
		EmployeeID: emp.ID,
		ManagerID:  managerID,
		Gaps:       gaps,
		PlanText:   buildPlanText(&emp, gaps),
		Items:      s.recommendItems(ctx, &emp, gaps),
		Status:     model.CoachingDraft,
	}
	if err := s.db.WithContext(ctx).Create(rec).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, rec.ID)
}

// diagnoseGaps finds weak capabilities from the latest assessment; falls back
// to low-scoring completed test tasks when no assessment exists.
func (s *CoachingService) diagnoseGaps(ctx context.Context, emp *model.Employee) []model.CoachingGap {
	var gaps []model.CoachingGap
	var assess model.Assessment
	if err := s.db.WithContext(ctx).Where("employee_id = ?", emp.ID).
		Order("assess_date desc").First(&assess).Error; err == nil {
		var scores []model.AssessmentScore
		s.db.WithContext(ctx).Where("assessment_id = ?", assess.ID).Find(&scores)
		for _, sc := range scores {
			if sc.Score < sc.TargetLevel {
				gaps = append(gaps, model.CoachingGap{
					Capability: sc.Capability,
					Score:      sc.Score,
					Target:     sc.TargetLevel,
					Evidence:   fmt.Sprintf("Assessment #%d (%s)", assess.ID, assess.AssessDate.Format("2006-01-02")),
				})
			}
		}
		return gaps
	}
	var tasks []model.LearningTask
	s.db.WithContext(ctx).Where("employee_id = ? AND task_type = ? AND status = ? AND score IS NOT NULL",
		emp.ID, model.TaskTest, model.TaskCompleted).Find(&tasks)
	for _, t := range tasks {
		if t.Score != nil && *t.Score < 70 {
			gaps = append(gaps, model.CoachingGap{
				Capability: t.Title,
				Score:      *t.Score,
				Target:     70,
				Evidence:   fmt.Sprintf("Test task #%d", t.ID),
			})
		}
	}
	return gaps
}

// hasLearningEvidence reports whether the employee has any scored learning
// data (assessments or graded test tasks) the analysis could draw on.
func (s *CoachingService) hasLearningEvidence(ctx context.Context, employeeID uint) bool {
	var n int64
	s.db.WithContext(ctx).Model(&model.Assessment{}).Where("employee_id = ?", employeeID).Count(&n)
	if n > 0 {
		return true
	}
	s.db.WithContext(ctx).Model(&model.LearningTask{}).
		Where("employee_id = ? AND task_type = ? AND status = ? AND score IS NOT NULL",
			employeeID, model.TaskTest, model.TaskCompleted).Count(&n)
	return n > 0
}

// recommendItems maps gaps to existing courses/materials of the employee's
// position. Only real resources are referenced.
func (s *CoachingService) recommendItems(ctx context.Context, emp *model.Employee, gaps []model.CoachingGap) []model.CoachingItem {
	var items []model.CoachingItem
	used := map[string]bool{}
	var courses []model.Course
	s.db.WithContext(ctx).Where("position_id = ?", emp.PositionID).Find(&courses)
	var materials []model.TrainingMaterial
	s.db.WithContext(ctx).Where("position_id = ?", emp.PositionID).Find(&materials)

	reason := func(g model.CoachingGap) string {
		return fmt.Sprintf("Weak in %s (%.0f/%.0f)", g.Capability, g.Score, g.Target)
	}
	for _, g := range gaps {
		if len(items) >= 3 {
			break
		}
		capLower := strings.ToLower(g.Capability)
		matched := false
		for _, c := range courses {
			key := fmt.Sprintf("course:%d", c.ID)
			if used[key] || !strings.Contains(strings.ToLower(c.Title), capLower) {
				continue
			}
			items = append(items, model.CoachingItem{Kind: "course", RefID: c.ID, Title: c.Title, Reason: reason(g), Accepted: true})
			used[key] = true
			matched = true
			break
		}
		if matched {
			continue
		}
		for _, m := range materials {
			key := fmt.Sprintf("material:%d", m.ID)
			if used[key] || !strings.Contains(strings.ToLower(m.Title), capLower) {
				continue
			}
			items = append(items, model.CoachingItem{Kind: "material", RefID: m.ID, Title: m.Title, Reason: reason(g), Accepted: true})
			used[key] = true
			break
		}
	}
	// Top up with remaining position courses so the manager always has
	// something concrete to assign.
	for _, c := range courses {
		if len(items) >= 3 {
			break
		}
		key := fmt.Sprintf("course:%d", c.ID)
		if used[key] {
			continue
		}
		items = append(items, model.CoachingItem{Kind: "course", RefID: c.ID, Title: c.Title, Reason: "Recommended for this position", Accepted: true})
		used[key] = true
	}
	return items
}

// buildPlanText renders a concrete, editable coaching plan from the gaps.
func buildPlanText(emp *model.Employee, gaps []model.CoachingGap) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s has %d learning gap(s) based on recent data:\n", emp.Name, len(gaps))
	for _, g := range gaps {
		fmt.Fprintf(&b, "- %s: %.0f/%.0f (%s)\n", g.Capability, g.Score, g.Target, g.Evidence)
	}
	b.WriteString("\nCoaching suggestions:\n")
	b.WriteString("1. Review the weak areas in the next 1:1 and agree on focus.\n")
	b.WriteString("2. Assign the checked supplement materials below; follow up in one week.\n")
	b.WriteString("3. Re-assess at the next stage checkpoint.")
	return b.String()
}

// List returns recommendations with pagination; ListOptions.EmployeeIDs scopes
// the result (used for manager department isolation).
func (s *CoachingService) List(ctx context.Context, opt ListOptions, employeeID uint) ([]model.CoachingRecommendation, int64, error) {
	var items []model.CoachingRecommendation
	query := s.db.WithContext(ctx).Model(&model.CoachingRecommendation{})
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
	if err := query.Preload("Employee").Order("created_at desc").
		Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Get returns one recommendation with its employee.
func (s *CoachingService) Get(ctx context.Context, id uint) (*model.CoachingRecommendation, error) {
	var rec model.CoachingRecommendation
	if err := s.db.WithContext(ctx).Preload("Employee").First(&rec, id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &rec, nil
}

// Update edits a draft recommendation's plan text and items.
func (s *CoachingService) Update(ctx context.Context, id uint, planText string, items []model.CoachingItem) (*model.CoachingRecommendation, error) {
	var rec model.CoachingRecommendation
	if err := s.db.WithContext(ctx).First(&rec, id).Error; err != nil {
		return nil, ErrNotFound
	}
	if rec.Status != model.CoachingDraft {
		return nil, fmt.Errorf("%w: recommendation already %s", ErrConflict, rec.Status)
	}
	if planText != "" {
		rec.PlanText = planText
	}
	if items != nil {
		rec.Items = items
	}
	if err := s.db.WithContext(ctx).Save(&rec).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Approve applies a draft recommendation: accepted items become supplement
// learning tasks in the employee's current stage. Idempotent: an already
// applied recommendation is returned unchanged.
func (s *CoachingService) Approve(ctx context.Context, id, managerID uint, planText string, items []model.CoachingItem) (*model.CoachingRecommendation, error) {
	var rec model.CoachingRecommendation
	if err := s.db.WithContext(ctx).First(&rec, id).Error; err != nil {
		return nil, ErrNotFound
	}
	if rec.Status != model.CoachingDraft {
		if rec.Status == model.CoachingApproved && rec.AppliedAt != nil {
			return s.Get(ctx, id)
		}
		return nil, fmt.Errorf("%w: recommendation already %s", ErrConflict, rec.Status)
	}
	if planText == "" {
		planText = rec.PlanText
	}
	if items == nil {
		items = rec.Items
	}

	var plan model.TrainingPlan
	if err := s.db.WithContext(ctx).Where("employee_id = ? AND status = ?", rec.EmployeeID, model.PlanInProgress).
		Order("created_at desc").First(&plan).Error; err != nil {
		return nil, fmt.Errorf("%w: employee has no active training plan", ErrValidation)
	}
	var stage model.TrainingStage
	if err := s.db.WithContext(ctx).Where("plan_id = ? AND stage_number = ?", plan.ID, plan.CurrentStage).
		First(&stage).Error; err != nil {
		if err2 := s.db.WithContext(ctx).Where("plan_id = ?", plan.ID).
			Order("stage_number desc").First(&stage).Error; err2 != nil {
			return nil, fmt.Errorf("%w: plan has no stages", ErrValidation)
		}
	}

	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			if !it.Accepted {
				continue
			}
			task := model.LearningTask{
				EmployeeID:    rec.EmployeeID,
				PlanID:        plan.ID,
				StageID:       stage.ID,
				Title:         "补充学习: " + it.Title,
				Description:   fmt.Sprintf("%s\n(From coaching recommendation #%d)", it.Reason, rec.ID),
				TaskType:      model.TaskLearning,
				StartDate:     now,
				DueDate:       now.Add(7 * 24 * time.Hour),
				Priority:      model.PriorityMedium,
				Status:        model.TaskPending,
				CreatedBy:     &managerID,
				CreatedSource: model.SourceAI,
				AIGenerated:   true,
			}
			if it.Kind == "course" {
				task.CourseID = &it.RefID
			}
			if it.Kind == "material" {
				task.MaterialID = &it.RefID
			}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
		}
		rec.Status = model.CoachingApproved
		rec.AppliedAt = &now
		rec.PlanText = planText
		rec.Items = items
		return tx.Save(&rec).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Reject marks a draft recommendation as rejected.
func (s *CoachingService) Reject(ctx context.Context, id uint) (*model.CoachingRecommendation, error) {
	var rec model.CoachingRecommendation
	if err := s.db.WithContext(ctx).First(&rec, id).Error; err != nil {
		return nil, ErrNotFound
	}
	if rec.Status != model.CoachingDraft {
		return nil, fmt.Errorf("%w: recommendation already %s", ErrConflict, rec.Status)
	}
	rec.Status = model.CoachingRejected
	if err := s.db.WithContext(ctx).Save(&rec).Error; err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}
