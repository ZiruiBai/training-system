package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AgentHandler exposes the AI assistant chat endpoint backed by G.AIOS.
type AgentHandler struct {
	agent   *service.AgentService
	dataSvc *service.AgentDataService
	db      *gorm.DB
}

// NewAgentHandler creates an AgentHandler.
func NewAgentHandler(agent *service.AgentService, dataSvc *service.AgentDataService, db *gorm.DB) *AgentHandler {
	return &AgentHandler{agent: agent, dataSvc: dataSvc, db: db}
}

type trainingChatReq struct {
	EmployeeID string `json:"employee_id"` // optional; manager/admin may target a specific employee
	Message    string `json:"message"`
	SessionID  string `json:"session_id"`
}

type trainingChatResp struct {
	SessionID string `json:"session_id"`
	Answer    string `json:"answer"`
}

// ChatAnswer handles POST /api/training/chat. The target employee identity is
// resolved from the authenticated session (never trusted from the client),
// the backend loads that employee's real data, and injects it into the agent.
func (h *AgentHandler) ChatAnswer(c *gin.Context) {
	var req trainingChatReq
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		badRequest(c, "message is required")
		return
	}

	// Resolve the target employee respecting row-level authorization.
	employeeID, err := h.resolveEmployeeID(c, req.EmployeeID)
	if err != nil {
		mapServiceError(c, err)
		return
	}

	// Backend loads the authoritative business data and injects it.
	contextBlob, err := h.dataSvc.EmployeeContext(c.Request.Context(), employeeID)
	if err != nil {
		mapServiceError(c, err)
		return
	}

	userID := "employee:" + strconv.FormatUint(uint64(employeeID), 10)
	result, err := h.agent.Chat(c.Request.Context(), userID, service.ChatRequest{
		EmployeeCode: req.EmployeeID,
		Message:      req.Message,
		SessionID:    req.SessionID,
		Context:      contextBlob,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAgentKeyMissing):
			fail(c, http.StatusServiceUnavailable, "AGENT_NOT_CONFIGURED", "AI assistant is not configured on the server", nil)
		case errors.Is(err, service.ErrAgentRequest):
			fail(c, http.StatusBadGateway, "AGENT_REQUEST_FAILED", "AI assistant is temporarily unavailable", nil)
		default:
			internal(c)
		}
		return
	}

	ok(c, trainingChatResp{SessionID: result.SessionID, Answer: result.Answer})
}

// resolveEmployeeID returns the authorized target employee id. The identity is
// taken from the session, not trusted from the client:
//   - employee: always resolves to self (ignores any client-supplied id)
//   - manager:  may target an employee they manage, otherwise self if they have one
//   - admin:    may target any employee by id/code
func (h *AgentHandler) resolveEmployeeID(c *gin.Context, idOrCode string) (uint, error) {
	p := middleware.GetPrincipal(c)
	if p == nil {
		return 0, service.ErrForbidden
	}

	// Employee: always self.
	if p.IsRole(model.RoleEmployee) {
		if p.EmployeeID == nil {
			return 0, service.ErrForbidden
		}
		return *p.EmployeeID, nil
	}

	// Admin: may target anyone; default to their own employee if present.
	if p.IsRole(model.RoleAdmin) {
		if strings.TrimSpace(idOrCode) == "" {
			if p.EmployeeID != nil {
				return *p.EmployeeID, nil
			}
			return 0, service.ErrValidation
		}
		return h.lookupEmployeeID(c, idOrCode)
	}

	// Manager: must target one of their own staff.
	if strings.TrimSpace(idOrCode) == "" {
		if p.EmployeeID != nil {
			return *p.EmployeeID, nil
		}
		return 0, service.ErrValidation
	}
	empID, err := h.lookupEmployeeID(c, idOrCode)
	if err != nil {
		return 0, err
	}
	var emp model.Employee
	if err := h.db.WithContext(c.Request.Context()).First(&emp, empID).Error; err != nil {
		return 0, mapGormErr(err)
	}
	if !service.ManagerCanAccessEmployee(c.Request.Context(), h.db, p.UserID, &emp) {
		return 0, service.ErrForbidden
	}
	return empID, nil
}

func (h *AgentHandler) lookupEmployeeID(c *gin.Context, idOrCode string) (uint, error) {
	if n, err := strconv.ParseUint(idOrCode, 10, 64); err == nil {
		var e model.Employee
		if err := h.db.WithContext(c.Request.Context()).First(&e, uint(n)).Error; err != nil {
			return 0, mapGormErr(err)
		}
		return e.ID, nil
	}
	var e model.Employee
	if err := h.db.WithContext(c.Request.Context()).Where("employee_code = ?", idOrCode).First(&e).Error; err != nil {
		return 0, mapGormErr(err)
	}
	return e.ID, nil
}

func mapGormErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return service.ErrNotFound
	}
	return err
}

// RegisterRoutes wires the chat endpoint.
func (h *AgentHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc) {
	train := r.Group("/training", auth)
	{
		train.POST("/chat", h.ChatAnswer)
	}
}
