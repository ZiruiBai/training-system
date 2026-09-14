package model

import (
	"time"

	"gorm.io/gorm"
)

// TrainingTemplateStatus describes a reusable template lifecycle.
type TrainingTemplateStatus string

const (
	TemplateDraft     TrainingTemplateStatus = "draft"
	TemplatePublished TrainingTemplateStatus = "published"
	TemplateArchived  TrainingTemplateStatus = "archived"
)

// TrainingTemplate is a reusable 30/60/90 training plan blueprint configured
// by HR. It holds a per-stage definition with sample tasks so that creating a
// plan for a new hire can be generated from a template.
type TrainingTemplate struct {
	ID           uint                   `gorm:"primaryKey" json:"id"`
	Title        string                 `gorm:"size:255;not null" json:"title"`
	Description  string                 `gorm:"type:text" json:"description"`
	PositionID   *uint                  `json:"position_id"`
	Position     *Position              `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	DepartmentID *uint                  `json:"department_id"`
	CycleDays    int                    `gorm:"not null;default:90" json:"cycle_days"`
	Status       TrainingTemplateStatus `gorm:"size:16;not null;default:'draft'" json:"status"`
	// AIOS-source provenance (reserved).
	CreatedSource CreatedSource           `gorm:"size:16;not null;default:'manual'" json:"created_source"`
	AIGenerated   bool                    `gorm:"not null;default:false" json:"ai_generated"`
	AIAgentID     string                  `gorm:"size:128" json:"ai_agent_id"`
	Stages        []TrainingTemplateStage `gorm:"foreignKey:TemplateID" json:"stages,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	DeletedAt     gorm.DeletedAt          `gorm:"index" json:"-"`
}

// TrainingTemplateStage is one segment (30/60/90) of a template.
type TrainingTemplateStage struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	TemplateID  uint   `gorm:"not null;index" json:"template_id"`
	StageNumber int    `gorm:"not null" json:"stage_number"`
	Name        string `gorm:"size:128;not null" json:"name"`
	Objectives  string `gorm:"type:text" json:"objectives"`
	// SampleTaskTitles are used to seed learning tasks when a plan is generated.
	SampleTaskTitles string         `gorm:"type:text" json:"sample_task_titles"`
	StartDayOffset   int            `gorm:"not null;default:0" json:"start_day_offset"`
	DurationDays     int            `gorm:"not null;default:30" json:"duration_days"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
