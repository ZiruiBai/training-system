package model

// AgentEmployeeProfile is the business-data profile exposed to the G.AIOS
// agent for a given employee.
type AgentEmployeeProfile struct {
	EmployeeID  uint   `json:"employee_id"`
	Name        string `json:"name"`
	Department  string `json:"department"`
	Position    string `json:"position"`
	JoinDate    string `json:"join_date"`
	CurrentStage int   `json:"current_stage"`
	Status      string `json:"status"`
}

// AgentCapabilityScore is one capability dimension in the agent learning data.
type AgentCapabilityScore struct {
	Capability  string  `json:"capability"`
	Score       float64 `json:"score"`
	TargetLevel float64 `json:"target_level"`
	Met         bool    `json:"met"`
}

// AgentAssessmentSummary is one past stage assessment.
type AgentAssessmentSummary struct {
	StageNumber    int     `json:"stage_number"`
	AssessDate     string  `json:"assess_date"`
	CompositeScore float64 `json:"composite_score"`
	GoalsMet       bool    `json:"goals_met"`
	Status         string  `json:"status"`
}

// AgentLearningData is the aggregated learning situation for an employee.
type AgentLearningData struct {
	EmployeeID      uint   `json:"employee_id"`
	PlanID          uint   `json:"plan_id"`
	PlanStatus      string `json:"plan_status"`
	CurrentStage    int    `json:"current_stage"`
	PlanProgress    float64 `json:"plan_progress"`
	TotalTasks      int    `json:"total_tasks"`
	CompletedTasks  int    `json:"completed_tasks"`
	IncompleteTasks int    `json:"incomplete_tasks"`
	OverdueTasks    int    `json:"overdue_tasks"`
	TotalCourses    int    `json:"total_courses"`
	CompletedCourses int   `json:"completed_courses"`
	CourseRate      float64 `json:"course_rate"`
	TestScores      []float64 `json:"test_scores"`
	AvgTestScore    float64 `json:"avg_test_score"`
	Capabilities    []AgentCapabilityScore `json:"capabilities"`
	Assessments     []AgentAssessmentSummary `json:"assessments"`
}

// AgentTask is one recent learning task for an employee.
type AgentTask struct {
	TaskID      uint    `json:"task_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	TaskType    string  `json:"task_type"`
	DueDate     string  `json:"due_date"`
	Status      string  `json:"status"`
	Score       *float64 `json:"score"`
}