package model

// AssessmentSnapshot is the aggregate payload for an employee's personal
// growth dashboard: latest assessment with scores, a composite-score trend,
// and stage summaries.
type AssessmentSnapshot struct {
	EmployeeID   uint                   `json:"employee_id"`
	Latest       *Assessment            `json:"latest"`
	Scores       []AssessmentScore      `json:"scores"`
	Trend        []AssessmentTrendPoint `json:"trend"`
	Stages       []AssessmentStageRow   `json:"stages"`
	Capabilities []CapabilitySummary    `json:"capabilities"`
}

// AssessmentTrendPoint is one point on the composite-score trend line.
type AssessmentTrendPoint struct {
	StageNumber    int     `json:"stage_number"`
	AssessDate     string  `json:"assess_date"`
	CompositeScore float64 `json:"composite_score"`
	GoalsMet       bool    `json:"goals_met"`
}

// AssessmentStageRow summarises one stage's assessment outcome.
type AssessmentStageRow struct {
	StageNumber    int     `json:"stage_number"`
	CompositeScore float64 `json:"composite_score"`
	Status         string  `json:"status"`
}

// CapabilitySummary is one capability dimension for the radar chart.
type CapabilitySummary struct {
	Capability  string  `json:"capability"`
	Score       float64 `json:"score"`
	TargetLevel float64 `json:"target_level"`
	Met         bool    `json:"met"`
}
