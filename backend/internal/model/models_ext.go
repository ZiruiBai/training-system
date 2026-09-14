package model

import (
	"time"

	"gorm.io/gorm"
)

// LearningRecord captures time spent and results on a task.
type LearningRecord struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	EmployeeID  uint           `gorm:"not null;index" json:"employee_id"`
	Employee    *Employee      `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	TaskID      uint           `gorm:"not null;index" json:"task_id"`
	Task        *LearningTask  `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Content     string         `gorm:"type:text" json:"content"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     *time.Time     `json:"end_time"`
	DurationMin int            `gorm:"not null;default:0" json:"duration_min"`
	Status      TaskStatus     `gorm:"size:16;not null;default:'pending'" json:"status"`
	TestScore   *float64       `json:"test_score"`
	SelfEval    string         `gorm:"size:1024" json:"self_eval"`
	ManagerEval string         `gorm:"size:1024" json:"manager_eval"`
	Outcome     string         `gorm:"type:text" json:"outcome"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// AssessmentStatus describes an assessment lifecycle.
type AssessmentStatus string

const (
	AssessmentDraft   AssessmentStatus = "draft"
	AssessmentDone    AssessmentStatus = "done"
	AssessmentPending AssessmentStatus = "pending"
)

// AssessmentSource indicates manual vs AI source.
type AssessmentSource string

const (
	AssessmentSourceManual AssessmentSource = "manual"
	AssessmentSourceAI     AssessmentSource = "ai"
)

// Assessment is a stage-based capability evaluation for an employee.
type Assessment struct {
	ID             uint             `gorm:"primaryKey" json:"id"`
	EmployeeID     uint             `gorm:"not null;index" json:"employee_id"`
	Employee       *Employee        `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	PlanID         uint             `gorm:"not null;index" json:"plan_id"`
	StageNumber    int              `gorm:"not null" json:"stage_number"`
	AssessDate     time.Time        `json:"assess_date"`
	AssessorID     *uint            `json:"assessor_id"`
	Assessor       *User            `gorm:"foreignKey:AssessorID" json:"assessor,omitempty"`
	CompositeScore float64          `json:"composite_score"`
	Strengths      string           `gorm:"type:text" json:"strengths"`
	Weaknesses     string           `gorm:"type:text" json:"weaknesses"`
	Improvement    string           `gorm:"type:text" json:"improvement"`
	GoalsMet       bool             `gorm:"not null;default:false" json:"goals_met"`
	Status         AssessmentStatus `gorm:"size:16;not null;default:'draft'" json:"status"`
	// AIOS-reserved fields.
	AssessmentSource AssessmentSource  `gorm:"size:16;not null;default:'manual'" json:"assessment_source"`
	AIGenerated      bool              `gorm:"not null;default:false" json:"ai_generated"`
	AIAgentID        string            `gorm:"size:128" json:"ai_agent_id"`
	Scores           []AssessmentScore `gorm:"foreignKey:AssessmentID" json:"scores,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `gorm:"index" json:"-"`
}

// AssessmentScore is one capability dimension within an assessment.
type AssessmentScore struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AssessmentID uint      `gorm:"not null;index" json:"assessment_id"`
	Capability   string    `gorm:"size:128;not null" json:"capability"`
	Score        float64   `gorm:"not null" json:"score"`
	TargetLevel  float64   `gorm:"not null;default:80" json:"target_level"`
	Weight       float64   `gorm:"not null;default:1" json:"weight"`
	Remark       string    `gorm:"size:512" json:"remark"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CourseStatus describes a course lifecycle.
type CourseStatus string

const (
	CourseDraft     CourseStatus = "draft"
	CoursePublished CourseStatus = "published"
	CourseOffline   CourseStatus = "offline"
)

// CourseCategory categorizes courses.
type CourseCategory string

const (
	CourseCategoryPolicy    CourseCategory = "policy"
	CourseCategoryPosition  CourseCategory = "position_base"
	CourseCategorySpecialty CourseCategory = "specialty"
	CourseCategoryTool      CourseCategory = "tool"
	CourseCategoryPractice  CourseCategory = "practice"
	CourseCategoryGeneral   CourseCategory = "general"
)

// Course is a training course resource.
type Course struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	Category      CourseCategory `gorm:"size:16;not null" json:"category"`
	PositionID    *uint          `json:"position_id"`
	Position      *Position      `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	Description   string         `gorm:"type:text" json:"description"`
	Objectives    string         `gorm:"type:text" json:"objectives"`
	Content       string         `gorm:"type:text" json:"content"`
	FilePath      string         `gorm:"size:512" json:"file_path"`
	Difficulty    int            `gorm:"not null;default:1" json:"difficulty"`
	EstimateHours float64        `json:"estimate_hours"`
	Status        CourseStatus   `gorm:"size:16;not null;default:'published'" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// MaterialCategory categorizes training materials.
type MaterialCategory string

const (
	MaterialPolicy    MaterialCategory = "policy"
	MaterialPosition  MaterialCategory = "position_base"
	MaterialSpecialty MaterialCategory = "specialty"
	MaterialTool      MaterialCategory = "tool"
	MaterialGeneral   MaterialCategory = "general"
)

// CoachingRecommendationStatus describes the review state of a coaching recommendation.
type CoachingRecommendationStatus string

const (
	CoachingDraft    CoachingRecommendationStatus = "draft"
	CoachingApproved CoachingRecommendationStatus = "approved"
	CoachingRejected CoachingRecommendationStatus = "rejected"
)

// CoachingGap is one diagnosed learning gap, stored as JSON on the recommendation.
type CoachingGap struct {
	Capability string  `json:"capability"`
	Score      float64 `json:"score"`
	Target     float64 `json:"target"`
	Evidence   string  `json:"evidence"`
}

// CoachingItem is one recommended supplement (existing course/material),
// stored as JSON on the recommendation.
type CoachingItem struct {
	Kind     string `json:"kind"` // course | material
	RefID    uint   `json:"ref_id"`
	Title    string `json:"title"`
	Reason   string `json:"reason"`
	Accepted bool   `json:"accepted"`
}

// CoachingRecommendation is a rule-engine generated coaching proposal that a
// manager reviews; approving it materializes supplement tasks for the employee.
type CoachingRecommendation struct {
	ID         uint                         `gorm:"primaryKey" json:"id"`
	EmployeeID uint                         `gorm:"not null;index" json:"employee_id"`
	Employee   *Employee                    `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	ManagerID  uint                         `gorm:"not null;index" json:"manager_id"`
	Manager    *User                        `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	Gaps       []CoachingGap                `gorm:"serializer:json" json:"gaps"`
	PlanText   string                       `gorm:"type:text" json:"plan_text"`
	Items      []CoachingItem               `gorm:"serializer:json" json:"items"`
	Status     CoachingRecommendationStatus `gorm:"size:16;not null;default:'draft'" json:"status"`
	AppliedAt  *time.Time                   `json:"applied_at"`
	CreatedAt  time.Time                    `json:"created_at"`
	UpdatedAt  time.Time                    `json:"updated_at"`
}

// TrainingMaterial is an uploaded learning asset.
type TrainingMaterial struct {
	ID          uint             `gorm:"primaryKey" json:"id"`
	Title       string           `gorm:"size:255;not null" json:"title"`
	FileType    string           `gorm:"size:16" json:"file_type"`
	PositionID  *uint            `json:"position_id"`
	Position    *Position        `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	CourseID    *uint            `json:"course_id"`
	Course      *Course          `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Category    MaterialCategory `gorm:"size:16;not null;default:'general'" json:"category"`
	Description string           `gorm:"type:text" json:"description"`
	FilePath    string           `gorm:"size:512" json:"file_path"`
	UploaderID  *uint            `json:"uploader_id"`
	Status      string           `gorm:"size:16;not null;default:'active'" json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-"`
}

// SopType categorizes SOP documents.
type SopType string

const (
	SopTypeHandbook   SopType = "handbook"
	SopTypeAttendance SopType = "attendance"
	SopTypeLeave      SopType = "leave"
	SopTypeBenefits   SopType = "benefits"
	SopTypeSOP        SopType = "sop"
	SopTypeWorkflow   SopType = "workflow"
	SopTypePolicy     SopType = "policy"
)

// SopDocument is a company policy or standard operating procedure.
type SopDocument struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"size:255;not null" json:"title"`
	SopType      SopType        `gorm:"size:16;not null" json:"sop_type"`
	DepartmentID *uint          `json:"department_id"`
	PositionID   *uint          `json:"position_id"`
	Version      string         `gorm:"size:32;not null;default:'1.0'" json:"version"`
	Content      string         `gorm:"type:text" json:"content"`
	FilePath     string         `gorm:"size:512" json:"file_path"`
	EffectiveAt  time.Time      `json:"effective_at"`
	Status       string         `gorm:"size:16;not null;default:'active'" json:"status"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// ReportType categorizes training reports.
type ReportType string

const (
	ReportWeekly  ReportType = "weekly"
	ReportMonthly ReportType = "monthly"
	Report30      ReportType = "report_30"
	Report60      ReportType = "report_60"
	Report90      ReportType = "report_90"
)

// TrainingReport is the final or periodic report for an employee.
type TrainingReport struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	EmployeeID     uint           `gorm:"not null;index" json:"employee_id"`
	Employee       *Employee      `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	PlanID         *uint          `json:"plan_id"`
	ReportType     ReportType     `gorm:"size:16;not null" json:"report_type"`
	Period         string         `gorm:"size:64" json:"period"`
	StageNumber    *int           `json:"stage_number"`
	Progress       float64        `json:"progress"`
	TaskCompletion float64        `json:"task_completion"`
	LearningHours  float64        `json:"learning_hours"`
	CompositeScore *float64       `json:"composite_score"`
	Capabilities   string         `gorm:"type:text" json:"capabilities"`
	Strengths      string         `gorm:"type:text" json:"strengths"`
	Weaknesses     string         `gorm:"type:text" json:"weaknesses"`
	Risks          string         `gorm:"type:text" json:"risks"`
	Suggestions    string         `gorm:"type:text" json:"suggestions"`
	ManagerEval    string         `gorm:"type:text" json:"manager_eval"`
	Content        string         `gorm:"type:text" json:"content"`
	AIGenerated    bool           `gorm:"not null;default:false" json:"ai_generated"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Session stores a server-side auth session.
type Session struct {
	ID        uint           `gorm:"primaryKey" json:"-"`
	Token     string         `gorm:"size:64;uniqueIndex;not null" json:"-"`
	UserID    uint           `gorm:"not null;index" json:"-"`
	ExpiresAt time.Time      `gorm:"not null" json:"-"`
	CreatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
