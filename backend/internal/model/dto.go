package model

import "time"

// PageMeta is the standard list pagination metadata.
type PageMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// ErrorEnvelope is the standard failure body.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries stable, machine-readable error information.
type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// SessionInfo is safe identity exposed to the frontend after login.
type SessionInfo struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Role       Role   `json:"role"`
	Display    string `json:"display_name"`
	Email      string `json:"email"`
	EmployeeID *uint  `json:"employee_id"`
}

// UserSummary is a lightweight user representation for dropdowns.
type UserSummary struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"display_name"`
	Role        Role   `json:"role"`
}

// DashboardOverview aggregates training metrics for a role.
type DashboardOverview struct {
	EmployeesTotal      int     `json:"employees_total"`
	EmployeesInTraining int     `json:"employees_in_training"`
	PlansTotal          int     `json:"plans_total"`
	TasksToday          int     `json:"tasks_today"`
	TasksTodayCompleted int     `json:"tasks_today_completed"`
	TasksTodayRate      float64 `json:"tasks_today_rate"`
	AvgProgress         float64 `json:"avg_progress"`
	PlanCompletionRate  float64 `json:"plan_completion_rate"`
}

// RiskItem is one flagged training risk.
type RiskItem struct {
	EmployeeID uint   `json:"employee_id"`
	Employee   string `json:"employee"`
	Department string `json:"department"`
	Position   string `json:"position"`
	RiskType   string `json:"risk_type"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
}

// EmployeeProgressRow is one row in the admin progress list.
type EmployeeProgressRow struct {
	EmployeeID    uint    `json:"employee_id"`
	Name          string  `json:"name"`
	Position      string  `json:"position"`
	Department    string  `json:"department"`
	CurrentStage  int     `json:"current_stage"`
	TrainingDays  int     `json:"training_days"`
	Progress      float64 `json:"progress"`
	RiskStatus    string  `json:"risk_status"`
	TrainingState string  `json:"training_state"`
}

// DepartmentTrainingRow aggregates training by department.
type DepartmentTrainingRow struct {
	DepartmentID  uint    `json:"department_id"`
	Department    string  `json:"department"`
	InTraining    int     `json:"in_training"`
	AvgProgress   float64 `json:"avg_progress"`
	TaskRate      float64 `json:"task_rate"`
	RiskEmployees int     `json:"risk_employees"`
}

// CapabilityDimension is one radar/dimension score.
type CapabilityDimension struct {
	Capability  string  `json:"capability"`
	Score       float64 `json:"score"`
	TargetLevel float64 `json:"target_level"`
	Weight      float64 `json:"weight"`
	Met         bool    `json:"met"`
}

// AssessmentAnalysis is the processed result of an assessment.
type AssessmentAnalysis struct {
	CompositeScore    float64               `json:"composite_score"`
	MetCapabilities   []string              `json:"met_capabilities"`
	UnmetCapabilities []string              `json:"unmet_capabilities"`
	WeakCapabilities  []CapabilityDimension `json:"weak_capabilities"`
	Dimensions        []CapabilityDimension `json:"dimensions"`
}

// AIGenerateResponse is the wrapper used by AIOS mock endpoints.
type AIGenerateResponse struct {
	Status  string         `json:"status"`
	Source  CreatedSource  `json:"source"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

// TimeNow is a small helper for deterministic times in tests.
func TimeNow() time.Time { return time.Now().UTC() }

// EmployeeWorkbench is the Employee home screen payload.
type EmployeeWorkbench struct {
	EmployeeID       uint                 `json:"employee_id"`
	Name             string               `json:"name"`
	Position         string               `json:"position"`
	Department       string               `json:"department"`
	HireDays         int                  `json:"hire_days"`
	CurrentStage     int                  `json:"current_stage"`
	StageName        string               `json:"stage_name"`
	Progress         float64              `json:"progress"`
	PlanStatus       string               `json:"plan_status"`
	TasksToday       int                  `json:"tasks_today"`
	TasksTodayDone   int                  `json:"tasks_today_done"`
	RecentRecords    []LearningRecordView `json:"recent_records"`
	Capabilities     []CapabilityView     `json:"capabilities"`
	WeakCapabilities []CapabilityView     `json:"weak_capabilities"`
	Recommendations  []RecommendationView `json:"recommendations"`
}

// LearningRecordView is a compact learning record row.
type LearningRecordView struct {
	ID          uint      `json:"id"`
	TaskTitle   string    `json:"task_title"`
	Content     string    `json:"content"`
	StartTime   time.Time `json:"start_time"`
	DurationMin int       `json:"duration_min"`
	Status      string    `json:"status"`
}

// CapabilityView is a capability score row (from latest assessment).
type CapabilityView struct {
	Capability  string  `json:"capability"`
	Score       float64 `json:"score"`
	TargetLevel float64 `json:"target_level"`
	Weight      float64 `json:"weight"`
	Met         bool    `json:"met"`
}

// RecommendationView is a suggested learning content item.
type RecommendationView struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
	URL    string `json:"url"`
}

// ManagerTeamRow is one employee row in the manager workbench.
type ManagerTeamRow struct {
	EmployeeID       uint     `json:"employee_id"`
	Name             string   `json:"name"`
	Position         string   `json:"position"`
	CurrentStage     int      `json:"current_stage"`
	Progress         float64  `json:"progress"`
	TaskRate         float64  `json:"task_rate"`
	RiskStatus       string   `json:"risk_status"`
	ConsecutiveMiss  int      `json:"consecutive_miss"`
	WeakCapabilities []string `json:"weak_capabilities"`
}

// ManagerWorkbench is the Manager home screen payload.
type ManagerWorkbench struct {
	NewHires           int              `json:"new_hires"`
	TasksTodayDone     int              `json:"tasks_today_done"`
	TasksTodayTotal    int              `json:"tasks_today_total"`
	TasksTodayRate     float64          `json:"tasks_today_rate"`
	AvgProgress        float64          `json:"avg_progress"`
	RiskCount          int              `json:"risk_count"`
	PendingPlans       int              `json:"pending_plans"`
	PendingAssessments int              `json:"pending_assessments"`
	Team               []ManagerTeamRow `json:"team"`
	Risks              []RiskItem       `json:"risks"`
	ToDos              []ToDoItem       `json:"to_dos"`
}

// ToDoItem is a manager action item.
type ToDoItem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// AdminCockpit is the HR/Admin home screen payload.
type AdminCockpit struct {
	StageCompletion    []StageCompletionRow    `json:"stage_completion"`
	PlanCompletionRate float64                 `json:"plan_completion_rate"`
	AssessmentPassRate float64                 `json:"assessment_pass_rate"`
	DepartmentRanking  []DepartmentTrainingRow `json:"department_ranking"`
	RiskStats          struct {
		NoPlan  int `json:"no_plan"`
		Overdue int `json:"overdue"`
		Lagging int `json:"lagging"`
		High    int `json:"high"`
	} `json:"risk_stats"`
	ResourceUsage ResourceUsage `json:"resource_usage"`
}

// StageCompletionRow reports how many completed/completed stage per stage 1..3.
type StageCompletionRow struct {
	StageNumber int     `json:"stage_number"`
	Name        string  `json:"name"`
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	Rate        float64 `json:"rate"`
}

// ResourceUsage summarizes resource counts.
type ResourceUsage struct {
	Courses   int `json:"courses"`
	Materials int `json:"materials"`
	Sops      int `json:"sops"`
}
