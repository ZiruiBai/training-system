package model

import (
	"time"

	"gorm.io/gorm"
)

// PositionCompetency links a capability requirement to a position.
// These data are consumed later by AIOS to auto-generate personalized plans.
type PositionCompetency struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	PositionID   uint           `gorm:"not null;index" json:"position_id"`
	Position     *Position      `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	TargetLevel  float64        `gorm:"not null;default:80" json:"target_level"`
	Weight       float64        `gorm:"not null;default:1" json:"weight"`
	Description  string         `gorm:"size:512" json:"description"`
	PassStandard string         `gorm:"type:text" json:"pass_standard"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// AllModels returns the model set used for SQLite schema bootstrap.
func AllModels() []any {
	return []any{
		&User{}, &Department{}, &Position{}, &PositionCompetency{}, &Employee{},
		&TrainingPlan{}, &TrainingStage{}, &LearningTask{}, &LearningRecord{},
		&Assessment{}, &AssessmentScore{}, &Course{}, &TrainingMaterial{},
		&SopDocument{}, &TrainingReport{}, &TrainingTemplate{}, &TrainingTemplateStage{},
		&CoachingRecommendation{},
		&Session{},
	}
}
