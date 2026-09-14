package handler

import (
	"errors"
	"net/http"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CoachingHandler exposes AI-coaching recommendation endpoints for managers.
type CoachingHandler struct {
	coaching *service.CoachingService
	db       *gorm.DB
}

// NewCoachingHandler creates a CoachingHandler.
func NewCoachingHandler(coaching *service.CoachingService, db *gorm.DB) *CoachingHandler {
	return &CoachingHandler{coaching: coaching, db: db}
}

// canAccessEmployee enforces department isolation: admins pass, managers only
// access employees in their own department (or direct reports).
func (h *CoachingHandler) canAccessEmployee(c *gin.Context, employeeID uint) bool {
	p := middleware.GetPrincipal(c)
	if p == nil {
		return false
	}
	if p.IsRole(model.RoleAdmin) {
		return true
	}
	if !p.IsRole(model.RoleManager) {
		return false
	}
	var e model.Employee
	if err := h.db.WithContext(c.Request.Context()).First(&e, employeeID).Error; err != nil {
		return false
	}
	return service.ManagerCanAccessEmployee(c.Request.Context(), h.db, p.UserID, &e)
}

type analyzeReq struct {
	EmployeeID uint `json:"employee_id"`
}

// Analyze generates a draft coaching recommendation for an employee.
func (h *CoachingHandler) Analyze(c *gin.Context) {
	var req analyzeReq
	if !bindJSON(c, &req) {
		return
	}
	if req.EmployeeID == 0 {
		mapServiceError(c, service.ErrValidation)
		return
	}
	if !h.canAccessEmployee(c, req.EmployeeID) {
		fail403(c)
		return
	}
	p := middleware.GetPrincipal(c)
	rec, err := h.coaching.Analyze(c.Request.Context(), req.EmployeeID, p.UserID)
	if err != nil {
		// 'No gaps' / 'no data' are normal analysis outcomes, not failures:
		// return distinct codes so the UI can show a friendly message.
		switch {
		case errors.Is(err, service.ErrNoGaps):
			fail(c, http.StatusBadRequest, "NO_GAPS", err.Error(), nil)
		case errors.Is(err, service.ErrNoLearningData):
			fail(c, http.StatusBadRequest, "NO_LEARNING_DATA", err.Error(), nil)
		default:
			mapServiceError(c, err)
		}
		return
	}
	created(c, rec)
}

// List returns recommendations; managers only see their own department's.
func (h *CoachingHandler) List(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		ids, err := service.ManagedEmployeeIDs(c.Request.Context(), h.db, p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		opt.EmployeeIDs = ids
	}
	items, total, err := h.coaching.List(c.Request.Context(), opt, employeeID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

// Get returns one recommendation with department isolation.
func (h *CoachingHandler) Get(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	rec, err := h.coaching.Get(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if !h.canAccessEmployee(c, rec.EmployeeID) {
		fail403(c)
		return
	}
	ok(c, rec)
}

type reviewReq struct {
	PlanText string               `json:"plan_text"`
	Items    []model.CoachingItem `json:"items"`
}

// Update edits a draft recommendation (plan text + item selection).
func (h *CoachingHandler) Update(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	rec, err := h.coaching.Get(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if !h.canAccessEmployee(c, rec.EmployeeID) {
		fail403(c)
		return
	}
	var req reviewReq
	if !bindJSON(c, &req) {
		return
	}
	updated, err := h.coaching.Update(c.Request.Context(), id, req.PlanText, req.Items)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, updated)
}

// Approve applies a draft recommendation: accepted items become supplement
// tasks for the employee. Body is optional; stored values are used otherwise.
func (h *CoachingHandler) Approve(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	rec, err := h.coaching.Get(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if !h.canAccessEmployee(c, rec.EmployeeID) {
		fail403(c)
		return
	}
	var req reviewReq
	_ = c.ShouldBindJSON(&req) // body optional
	p := middleware.GetPrincipal(c)
	approved, err := h.coaching.Approve(c.Request.Context(), id, p.UserID, req.PlanText, req.Items)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, approved)
}

// Reject marks a draft recommendation as rejected.
func (h *CoachingHandler) Reject(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	rec, err := h.coaching.Get(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if !h.canAccessEmployee(c, rec.EmployeeID) {
		fail403(c)
		return
	}
	rejected, err := h.coaching.Reject(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, rejected)
}

// RegisterRoutes wires coaching endpoints.
func (h *CoachingHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	g := r.Group("/coaching", auth, role(model.RoleManager, model.RoleAdmin))
	{
		g.POST("/analyze", h.Analyze)
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.POST("/:id/approve", h.Approve)
		g.POST("/:id/reject", h.Reject)
	}
}
