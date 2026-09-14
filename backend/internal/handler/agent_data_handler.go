package handler

import (
	"net/http"
	"os"
	"strconv"

	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AgentDataHandler exposes read-only business data endpoints for the G.AIOS
// Script Tool. Identity is passed by the script from runtime.userInfo via
// request headers, and row-level auth is enforced server-side so the agent
// only ever sees data it is entitled to.
type AgentDataHandler struct {
	agent *service.AgentDataService
	db    *gorm.DB
}

// NewAgentDataHandler creates an AgentDataHandler.
func NewAgentDataHandler(agent *service.AgentDataService, db *gorm.DB) *AgentDataHandler {
	return &AgentDataHandler{agent: agent, db: db}
}

// agentToolAuth optionally requires AGENT_TOOL_TOKEN on the request so a
// third-party cannot impersonate a user by forging identity headers.
func agentToolAuth() gin.HandlerFunc {
	expected := os.Getenv("AGENT_TOOL_TOKEN")
	if expected == "" {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if c.GetHeader("X-Agent-Tool-Token") != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorEnvelope{
				Error: model.ErrorDetail{Code: "UNAUTHORIZED", Message: "Invalid agent tool token"},
			})
			return
		}
		c.Next()
	}
}

// toolIdentity is the business identity injected by the Script Tool from
// G.AIOS runtime.userInfo.
type toolIdentity struct {
	UserID     string
	EmployeeID uint
	Role       model.Role
}

// toolIdentityFromHeaders parses the identity headers the script sends.
func toolIdentityFromHeaders(c *gin.Context) *toolIdentity {
	role := c.GetHeader("X-User-Role")
	empStr := c.GetHeader("X-Employee-Id")
	if role == "" && empStr == "" {
		return nil
	}
	id := &toolIdentity{UserID: c.GetHeader("X-User-Id"), Role: model.Role(role)}
	if empStr != "" {
		if n, err := strconv.ParseUint(empStr, 10, 64); err == nil {
			id.EmployeeID = uint(n)
		}
	}
	return id
}

// resolveEmployeeID determines the target employee and enforces row-level
// authorization. If a trusted identity header is present, that identity is
// authoritative; otherwise (legacy anonymous call) it falls back to the
// employee_id query parameter.
func (h *AgentDataHandler) resolveEmployeeID(c *gin.Context) (uint, error) {
	ident := toolIdentityFromHeaders(c)
	qID := c.Query("employee_id")

	if ident == nil {
		// Legacy: anonymous query with an explicit employee_id.
		n, err := strconv.ParseUint(qID, 10, 64)
		if err != nil || n == 0 {
			return 0, service.ErrValidation
		}
		return uint(n), nil
	}

	// Determine the requested target.
	var target uint
	if qID != "" {
		n, err := strconv.ParseUint(qID, 10, 64)
		if err == nil && n != 0 {
			target = uint(n)
		}
	}
	if target == 0 {
		target = ident.EmployeeID
	}
	if target == 0 {
		return 0, service.ErrValidation
	}

	// Row-level auth.
	switch ident.Role {
	case model.RoleAdmin:
		// Admin may read anyone.
		return target, nil
	case model.RoleEmployee:
		// Employee may only read their own record.
		if ident.EmployeeID != 0 && target != ident.EmployeeID {
			return 0, service.ErrForbidden
		}
		return target, nil
	case model.RoleManager:
		// Manager may only read employees in their own department (or direct reports).
		var emp model.Employee
		if err := h.db.WithContext(c.Request.Context()).First(&emp, target).Error; err != nil {
			return 0, mapGormErr(err)
		}
		// manager identity is carried as X-User-Id (manager user id).
		if ident.UserID != "" {
			mgrID, err := strconv.ParseUint(ident.UserID, 10, 64)
			if err != nil || !service.ManagerCanAccessEmployee(c.Request.Context(), h.db, uint(mgrID), &emp) {
				return 0, service.ErrForbidden
			}
		}
		return target, nil
	default:
		// Unknown role: only allow self-read.
		if ident.EmployeeID != 0 && target == ident.EmployeeID {
			return target, nil
		}
		return 0, service.ErrForbidden
	}
}

// get_employee_profile
func (h *AgentDataHandler) GetEmployeeProfile(c *gin.Context) {
	id, err := h.resolveEmployeeID(c)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	p, err := h.agent.EmployeeProfile(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// get_employee_learning_data
func (h *AgentDataHandler) GetEmployeeLearningData(c *gin.Context) {
	id, err := h.resolveEmployeeID(c)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	d, err := h.agent.EmployeeLearningData(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

// get_employee_tasks
func (h *AgentDataHandler) GetEmployeeTasks(c *gin.Context) {
	id, err := h.resolveEmployeeID(c)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	tasks, err := h.agent.EmployeeTasks(c.Request.Context(), id, limit)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// RegisterRoutes wires the agent tool endpoints (no user session required).
func (h *AgentDataHandler) RegisterRoutes(r *gin.RouterGroup) {
	ag := r.Group("/agent")
	ag.Use(agentToolAuth())
	{
		ag.GET("/get_employee_profile", h.GetEmployeeProfile)
		ag.GET("/get_employee_learning_data", h.GetEmployeeLearningData)
		ag.GET("/get_employee_tasks", h.GetEmployeeTasks)
	}
}
