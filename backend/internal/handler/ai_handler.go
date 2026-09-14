package handler

import (
	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/gin-gonic/gin"
)

// AIHandler exposes reserved AIOS endpoints that currently return mock
// contracts. These are placeholders so the API surface remains stable before
// an AIOS agent is wired in.
type AIHandler struct{}

// NewAIHandler creates an AIHandler.
func NewAIHandler() *AIHandler { return &AIHandler{} }

func aiResponse(status, message string) model.AIGenerateResponse {
	return model.AIGenerateResponse{Status: status, Source: model.SourceAI, Message: message}
}

// GenerateTrainingPlan is a mock of the AIOS plan-generation contract.
func (h *AIHandler) GenerateTrainingPlan(c *gin.Context) {
	ok(c, aiResponse("ready", "Training plan generation is reserved for AIOS. Provide employee_id and position to receive a generated 30/60/90 plan."))
}

// GenerateTasks is a mock of the AIOS task-generation contract.
func (h *AIHandler) GenerateTasks(c *gin.Context) {
	ok(c, aiResponse("ready", "Daily task generation is reserved for AIOS. Provide plan_id and stage_id."))
}

// AnalyzeGrowth is a mock of the AIOS growth analysis contract.
func (h *AIHandler) AnalyzeGrowth(c *gin.Context) {
	ok(c, aiResponse("ready", "Employee growth analysis is reserved for AIOS. Provide employee_id."))
}

// GenerateReport is a mock of the AIOS report-generation contract.
func (h *AIHandler) GenerateReport(c *gin.Context) {
	ok(c, aiResponse("ready", "Training report generation is reserved for AIOS. Provide employee_id and report_type."))
}

// GenerateRecommendations is a mock of the AIOS recommendation contract.
func (h *AIHandler) GenerateRecommendations(c *gin.Context) {
	ok(c, aiResponse("ready", "Learning recommendations are reserved for AIOS. Provide employee_id and weaknesses."))
}

// RegisterRoutes wires the AIOS placeholder endpoints (admin/manager only).
func (h *AIHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	ai := r.Group("/ai", auth, role(model.RoleAdmin, model.RoleManager))
	{
		ai.POST("/training-plan/generate", h.GenerateTrainingPlan)
		ai.POST("/tasks/generate", h.GenerateTasks)
		ai.POST("/assessment/analyze", h.AnalyzeGrowth)
		ai.POST("/report/generate", h.GenerateReport)
		ai.POST("/recommendations/generate", h.GenerateRecommendations)
	}
}
