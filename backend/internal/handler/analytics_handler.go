package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// AnalyticsHandler exposes dashboard, risks, assessments and reports.
type AnalyticsHandler struct {
	analytics *service.AnalyticsService
}

// NewAnalyticsHandler creates an AnalyticsHandler.
func NewAnalyticsHandler(analytics *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

// Dashboard returns role-aware metrics.
func (h *AnalyticsHandler) Dashboard(c *gin.Context) {
	p := middleware.GetPrincipal(c)
	if p == nil {
		internal(c)
		return
	}
	o, err := h.analytics.Dashboard(c.Request.Context(), p)
	if err != nil {
		internal(c)
		return
	}
	ok(c, o)
}

// RoleHome returns the role-specific home screen payload.
func (h *AnalyticsHandler) RoleHome(c *gin.Context) {
	p := middleware.GetPrincipal(c)
	if p == nil {
		fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required", nil)
		return
	}
	switch p.Role {
	case model.RoleEmployee:
		if p.EmployeeID == nil {
			fail(c, http.StatusBadRequest, "BAD_REQUEST", "Employee account has no linked employee record", nil)
			return
		}
		w, err := h.analytics.EmployeeWorkbench(c.Request.Context(), *p.EmployeeID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		ok(c, w)
	case model.RoleManager:
		w, err := h.analytics.ManagerWorkbench(c.Request.Context(), p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		ok(c, w)
	default:
		w, err := h.analytics.AdminCockpit(c.Request.Context())
		if err != nil {
			mapServiceError(c, err)
			return
		}
		ok(c, w)
	}
}

func (h *AnalyticsHandler) EmployeeProgress(c *gin.Context) {
	rows, err := h.analytics.EmployeeProgressRows(c.Request.Context())
	if err != nil {
		internal(c)
		return
	}
	ok(c, rows)
}

func (h *AnalyticsHandler) DepartmentTraining(c *gin.Context) {
	rows, err := h.analytics.DepartmentTrainingRows(c.Request.Context())
	if err != nil {
		internal(c)
		return
	}
	ok(c, rows)
}

func (h *AnalyticsHandler) Risks(c *gin.Context) {
	items, err := h.analytics.Risks(c.Request.Context())
	if err != nil {
		internal(c)
		return
	}
	ok(c, items)
}

// --- Assessments ---

func (h *AnalyticsHandler) ListAssessments(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	planID := queryUint(c, "plan_id")
	// Managers can only list assessments of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		ids, err := h.analytics.ManagedEmployeeIDs(c.Request.Context(), p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		opt.EmployeeIDs = ids
	}
	items, total, err := h.analytics.ListAssessments(c.Request.Context(), opt, employeeID, planID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *AnalyticsHandler) GetAssessment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	a, err := h.analytics.GetAssessment(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	// Managers can only view assessments of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		if !h.analytics.ManagerCanAccessEmployeeID(c.Request.Context(), p.UserID, a.EmployeeID) {
			fail403(c)
			return
		}
	}
	ok(c, a)
}

func (h *AnalyticsHandler) AnalyzeAssessment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	an, err := h.analytics.AnalyzeAssessment(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, an)
}

func (h *AnalyticsHandler) MyGrowth(c *gin.Context) {
	p := middleware.GetPrincipal(c)
	if p == nil {
		fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", nil)
		return
	}

	var target uint
	if p.IsRole(model.RoleEmployee) {
		if p.EmployeeID == nil {
			fail(c, http.StatusForbidden, "FORBIDDEN", "No linked employee", nil)
			return
		}
		target = *p.EmployeeID
	} else {
		if v := c.Query("employee_id"); v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				target = uint(n)
			}
		}
		if target == 0 {
			if p.EmployeeID != nil {
				target = *p.EmployeeID
			} else {
				fail(c, http.StatusBadRequest, "BAD_REQUEST", "employee_id is required", nil)
				return
			}
		}
	}

	snap, err := h.analytics.MyGrowth(c.Request.Context(), target)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, snap)
}

func (h *AnalyticsHandler) CreateAssessment(c *gin.Context) {
	var body struct {
		EmployeeID  uint                    `json:"employee_id"`
		PlanID      uint                    `json:"plan_id"`
		StageNumber int                     `json:"stage_number"`
		AssessDate  *timeFromJSON           `json:"assess_date"`
		Strengths   string                  `json:"strengths"`
		Weaknesses  string                  `json:"weaknesses"`
		Improvement string                  `json:"improvement"`
		GoalsMet    bool                    `json:"goals_met"`
		Scores      []model.AssessmentScore `json:"scores"`
	}
	if !bindJSON(c, &body) {
		return
	}
	a := &model.Assessment{
		EmployeeID: body.EmployeeID, PlanID: body.PlanID, StageNumber: body.StageNumber,
		Strengths: body.Strengths, Weaknesses: body.Weaknesses,
		Improvement: body.Improvement, GoalsMet: body.GoalsMet, Status: model.AssessmentDone,
		AssessmentSource: model.AssessmentSourceManual,
	}
	if body.AssessDate != nil {
		a.AssessDate = body.AssessDate.Time()
	} else {
		a.AssessDate = time.Now().UTC()
	}
	if p := middleware.GetPrincipal(c); p != nil {
		a.AssessorID = &p.UserID
	}
	a, err := h.analytics.CreateAssessment(c.Request.Context(), a, body.Scores)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, a)
}

func (h *AnalyticsHandler) UpdateAssessment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var body struct {
		StageNumber int                     `json:"stage_number"`
		AssessDate  *timeFromJSON           `json:"assess_date"`
		Strengths   string                  `json:"strengths"`
		Weaknesses  string                  `json:"weaknesses"`
		Improvement string                  `json:"improvement"`
		GoalsMet    bool                    `json:"goals_met"`
		Scores      []model.AssessmentScore `json:"scores"`
	}
	if !bindJSON(c, &body) {
		return
	}
	a := &model.Assessment{
		StageNumber: body.StageNumber, Strengths: body.Strengths, Weaknesses: body.Weaknesses,
		Improvement: body.Improvement, GoalsMet: body.GoalsMet,
	}
	if body.AssessDate != nil {
		a.AssessDate = body.AssessDate.Time()
	}
	a, err := h.analytics.UpdateAssessment(c.Request.Context(), id, a, body.Scores)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, a)
}

// --- Reports ---

func (h *AnalyticsHandler) ListReports(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	// Employees can only list their own reports.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleEmployee) && p.EmployeeID != nil {
		employeeID = *p.EmployeeID
	}
	// Managers can only list reports of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		ids, err := h.analytics.ManagedEmployeeIDs(c.Request.Context(), p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		opt.EmployeeIDs = ids
	}
	items, total, err := h.analytics.ListReports(c.Request.Context(), opt, employeeID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *AnalyticsHandler) GetReport(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	r, err := h.analytics.GetReport(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	// Managers can only view reports of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		if !h.analytics.ManagerCanAccessEmployeeID(c.Request.Context(), p.UserID, r.EmployeeID) {
			fail403(c)
			return
		}
	}
	ok(c, r)
}

func (h *AnalyticsHandler) CreateReport(c *gin.Context) {
	var body model.TrainingReport
	if !bindJSON(c, &body) {
		return
	}
	r, err := h.analytics.CreateReport(c.Request.Context(), &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, r)
}

func (h *AnalyticsHandler) DeleteReport(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.analytics.DeleteReport(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// RegisterRoutes wires analytics endpoints.
func (h *AnalyticsHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	dashboard := r.Group("/dashboard", auth)
	{
		dashboard.GET("/role", h.RoleHome)
		dashboard.GET("/overview", h.Dashboard)
		dashboard.GET("/employee-progress", role(model.RoleAdmin, model.RoleManager), h.EmployeeProgress)
		dashboard.GET("/department-training", role(model.RoleAdmin), h.DepartmentTraining)
		dashboard.GET("/risks", role(model.RoleAdmin, model.RoleManager), h.Risks)
	}
	assessments := r.Group("/assessments", auth)
	{
		assessments.GET("", h.ListAssessments)
		assessments.GET("/me", h.MyGrowth)
		assessments.GET("/:id", h.GetAssessment)
		assessments.GET("/:id/analyze", h.AnalyzeAssessment)
		assessments.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateAssessment)
		assessments.PUT("/:id", role(model.RoleManager, model.RoleAdmin), h.UpdateAssessment)
	}
	reports := r.Group("/reports", auth)
	{
		reports.GET("", h.ListReports)
		reports.GET("/:id", h.GetReport)
		reports.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateReport)
		reports.DELETE("/:id", role(model.RoleAdmin), h.DeleteReport)
	}
}
