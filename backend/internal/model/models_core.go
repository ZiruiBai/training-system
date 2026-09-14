package model

import (
	"time"

	"gorm.io/gorm"
)

// Role is a system user role.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

// User represents an authenticated account.
type User struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Username    string         `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email       string         `gorm:"size:128;uniqueIndex" json:"email"`
	Password    string         `gorm:"size:255;not null" json:"-"`
	DisplayName string         `gorm:"size:128" json:"display_name"`
	Role        Role           `gorm:"size:16;not null;index" json:"role"`
	EmployeeID  *uint          `json:"employee_id"`
	Active      bool           `gorm:"not null;default:true" json:"active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Department represents an organizational unit.
type Department struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string         `gorm:"size:512" json:"description"`
	ManagerID   *uint          `json:"manager_id"`
	Manager     *User          `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Position represents a job role.
type Position struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"size:128;not null" json:"name"`
	DepartmentID     uint           `gorm:"not null;index" json:"department_id"`
	Department       *Department    `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	Description      string         `gorm:"size:512" json:"description"`
	Responsibilities string         `gorm:"type:text" json:"responsibilities"`
	Requirement      string         `gorm:"type:text" json:"requirement"`
	CoreAbilities    string         `gorm:"type:text" json:"core_abilities"`
	CycleDays        int            `gorm:"not null;default:90" json:"cycle_days"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// SkillLevel is an ability grade.
type SkillLevel string

const (
	LevelJunior SkillLevel = "junior"
	LevelMiddle SkillLevel = "middle"
	LevelSenior SkillLevel = "senior"
)

// TrainingStatus describes an employee's training state.
type TrainingStatus string

const (
	StatusNotStarted TrainingStatus = "not_started"
	StatusInProgress TrainingStatus = "in_progress"
	StatusCompleted  TrainingStatus = "completed"
	StatusPaused     TrainingStatus = "paused"
)

// Employee represents a new hire under training.
type Employee struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	EmployeeCode   string         `gorm:"size:32;uniqueIndex;not null" json:"employee_code"`
	Name           string         `gorm:"size:128;not null" json:"name"`
	Email          string         `gorm:"size:128;uniqueIndex;not null" json:"email"`
	Phone          string         `gorm:"size:32" json:"phone"`
	DepartmentID   uint           `gorm:"index" json:"department_id"`
	Department     *Department    `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	PositionID     uint           `gorm:"index" json:"position_id"`
	Position       *Position      `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	ManagerID      *uint          `json:"manager_id"`
	Manager        *User          `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	HireDate       time.Time      `json:"hire_date"`
	WorkExperience int            `json:"work_experience"`
	SkillLevel     SkillLevel     `gorm:"size:16;not null;default:'junior'" json:"skill_level"`
	CurrentStage   int            `gorm:"not null;default:1" json:"current_stage"`
	TrainingStatus TrainingStatus `gorm:"size:16;not null;default:'not_started'" json:"training_status"`
	Progress       float64        `gorm:"not null;default:0" json:"progress"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// PlanStatus describes a training plan lifecycle.
type PlanStatus string

const (
	PlanDraft      PlanStatus = "draft"
	PlanPending    PlanStatus = "pending"
	PlanPublished  PlanStatus = "published"
	PlanInProgress PlanStatus = "in_progress"
	PlanCompleted  PlanStatus = "completed"
	PlanPaused     PlanStatus = "paused"
)

// CreatedSource indicates manual vs AI creation.
type CreatedSource string

const (
	SourceManual CreatedSource = "manual"
	SourceAI     CreatedSource = "ai"
)

// TrainingPlan is the core document scoping stages and tasks for an employee.
type TrainingPlan struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	EmployeeID   uint       `gorm:"not null;index" json:"employee_id"`
	Employee     *Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      time.Time  `json:"end_date"`
	CycleDays    int        `gorm:"not null;default:90" json:"cycle_days"`
	CurrentStage int        `gorm:"not null;default:1" json:"current_stage"`
	Progress     float64    `gorm:"not null;default:0" json:"progress"`
	Status       PlanStatus `gorm:"size:16;not null;default:'draft';index" json:"status"`
	ReviewedBy   *uint      `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	// AIOS-reserved fields.
	CreatedBy      *uint           `json:"created_by"`
	CreatedSource  CreatedSource   `gorm:"size:16;not null;default:'manual'" json:"created_source"`
	AIGenerated    bool            `gorm:"not null;default:false" json:"ai_generated"`
	AIAgentID      string          `gorm:"size:128" json:"ai_agent_id"`
	AISessionID    string          `gorm:"size:128" json:"ai_session_id"`
	AIWorkflowID   string          `gorm:"size:128" json:"ai_workflow_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      gorm.DeletedAt  `gorm:"index" json:"-"`
	TrainingStages []TrainingStage `gorm:"foreignKey:PlanID" json:"training_stages,omitempty"`
}

// StageStatus describes a training stage state.
type StageStatus string

const (
	StageNotStarted StageStatus = "not_started"
	StageInProgress StageStatus = "in_progress"
	StageCompleted  StageStatus = "completed"
)

// TrainingStage is a 30/60/90 segment of a plan.
type TrainingStage struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PlanID      uint           `gorm:"not null;index" json:"plan_id"`
	Plan        *TrainingPlan  `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
	StageNumber int            `gorm:"not null" json:"stage_number"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Objectives  string         `gorm:"type:text" json:"objectives"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     time.Time      `json:"end_date"`
	Status      StageStatus    `gorm:"size:16;not null;default:'not_started'" json:"status"`
	Progress    float64        `gorm:"not null;default:0" json:"progress"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TaskType enumerates task categories.
type TaskType string

const (
	TaskLearning TaskType = "learning"
	TaskReading  TaskType = "reading"
	TaskVideo    TaskType = "video"
	TaskPractice TaskType = "practice"
	TaskHomework TaskType = "homework"
	TaskTest     TaskType = "test"
	TaskProject  TaskType = "project"
	TaskReview   TaskType = "review"
)

// QuizQuestion is a single multiple-choice question used for test tasks.
// Answer is the 1-based index of the correct option.
type QuizQuestion struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

// TaskStatus describes a task lifecycle.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskOverdue    TaskStatus = "overdue"
)

// Priority describes urgency.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// LearningTask is a discrete learning assignment under a stage.
type LearningTask struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	EmployeeID    uint           `gorm:"not null;index" json:"employee_id"`
	PlanID        uint           `gorm:"not null;index" json:"plan_id"`
	StageID       uint           `gorm:"index" json:"stage_id"`
	Stage         *TrainingStage `gorm:"foreignKey:StageID" json:"stage,omitempty"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	TaskType      TaskType       `gorm:"size:16;not null" json:"task_type"`
	CourseID      *uint          `json:"course_id"`
	MaterialID    *uint          `json:"material_id"`
	StartDate     time.Time      `json:"start_date"`
	DueDate       time.Time      `json:"due_date"`
	EstimateHours float64        `json:"estimate_hours"`
	Priority      Priority       `gorm:"size:8;not null;default:'medium'" json:"priority"`
	Status        TaskStatus     `gorm:"size:16;not null;default:'pending';index" json:"status"`
	CompletedAt   *time.Time     `json:"completed_at"`
	Score         *float64       `json:"score"`
	Outcome       string         `gorm:"type:text" json:"outcome"`
	Quiz          []QuizQuestion `gorm:"serializer:json" json:"quiz"`
	Remark        string         `gorm:"size:512" json:"remark"`
	CreatedBy     *uint          `json:"created_by"`
	CreatedSource CreatedSource  `gorm:"size:16;not null;default:'manual'" json:"created_source"`
	AIGenerated   bool           `gorm:"not null;default:false" json:"ai_generated"`
	AIAgentID     string         `gorm:"size:128" json:"ai_agent_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
