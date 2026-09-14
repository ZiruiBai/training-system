package handler

import (
	"time"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// TemplateHandler exposes customizable training templates.
type TemplateHandler struct {
	templates *service.TemplateService
}

// NewTemplateHandler creates a TemplateHandler.
func NewTemplateHandler(templates *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{templates: templates}
}

func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	positionID := queryUint(c, "position_id")
	status := c.Query("status")
	items, total, err := h.templates.ListTemplates(c.Request.Context(), opt, positionID, status)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	item, err := h.templates.GetTemplate(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, item)
}

type templateUpsert struct {
	Title        string                       `json:"title"`
	Description  string                       `json:"description"`
	PositionID   *uint                        `json:"position_id"`
	DepartmentID *uint                        `json:"department_id"`
	CycleDays    int                          `json:"cycle_days"`
	Status       model.TrainingTemplateStatus `json:"status"`
	Stages       []templateStageUpsert        `json:"stages"`
}

type templateStageUpsert struct {
	StageNumber      int    `json:"stage_number"`
	Name             string `json:"name"`
	Objectives       string `json:"objectives"`
	SampleTaskTitles string `json:"sample_task_titles"`
	StartDayOffset   int    `json:"start_day_offset"`
	DurationDays     int    `json:"duration_days"`
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var req templateUpsert
	if !bindJSON(c, &req) {
		return
	}
	t := &model.TrainingTemplate{
		Title: req.Title, Description: req.Description, PositionID: req.PositionID,
		DepartmentID: req.DepartmentID, CycleDays: req.CycleDays, Status: req.Status,
	}
	stages := templateStagesFrom(req.Stages)
	item, err := h.templates.CreateTemplate(c.Request.Context(), t, stages)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, item)
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req templateUpsert
	if !bindJSON(c, &req) {
		return
	}
	t := &model.TrainingTemplate{
		Title: req.Title, Description: req.Description, PositionID: req.PositionID,
		DepartmentID: req.DepartmentID, CycleDays: req.CycleDays, Status: req.Status,
	}
	stages := templateStagesFrom(req.Stages)
	item, err := h.templates.UpdateTemplate(c.Request.Context(), id, t, stages)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, item)
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.templates.DeleteTemplate(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

type createPlanFromTemplateReq struct {
	EmployeeID uint   `json:"employee_id"`
	TemplateID uint   `json:"template_id"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

// CreatePlanFromTemplate creates a draft plan with stages + seed tasks.
func (h *TemplateHandler) CreatePlanFromTemplate(c *gin.Context) {
	var req createPlanFromTemplateReq
	if !bindJSON(c, &req) {
		return
	}
	if req.EmployeeID == 0 || req.TemplateID == 0 {
		badRequest(c, "employee_id and template_id are required")
		return
	}
	plan := &model.TrainingPlan{
		EmployeeID: req.EmployeeID, CycleDays: 90,
	}
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			plan.StartDate = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			plan.EndDate = t
		}
	}
	if plan.StartDate.IsZero() {
		plan.StartDate = time.Now().UTC()
	}
	if plan.EndDate.IsZero() {
		plan.EndDate = plan.StartDate.AddDate(0, 0, 89)
	}
	var createdBy *uint
	if p := middleware.GetPrincipal(c); p != nil {
		createdBy = &p.UserID
	}
	item, err := h.templates.CreatePlanFromTemplate(c.Request.Context(), plan, req.TemplateID, createdBy)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, item)
}

func (h *TemplateHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	templates := r.Group("/templates", auth)
	{
		templates.GET("", role(model.RoleAdmin, model.RoleManager), h.ListTemplates)
		templates.GET("/:id", role(model.RoleAdmin, model.RoleManager), h.GetTemplate)
		templates.POST("", role(model.RoleAdmin), h.CreateTemplate)
		templates.PUT("/:id", role(model.RoleAdmin), h.UpdateTemplate)
		templates.DELETE("/:id", role(model.RoleAdmin), h.DeleteTemplate)
		templates.POST("/:id/create-plan", role(model.RoleAdmin, model.RoleManager), h.CreatePlanFromTemplate)
	}
}

func templateStagesFrom(req []templateStageUpsert) []model.TrainingTemplateStage {
	out := make([]model.TrainingTemplateStage, 0, len(req))
	for _, s := range req {
		out = append(out, model.TrainingTemplateStage{
			StageNumber: s.StageNumber, Name: s.Name, Objectives: s.Objectives,
			SampleTaskTitles: s.SampleTaskTitles, StartDayOffset: s.StartDayOffset,
			DurationDays: s.DurationDays,
		})
	}
	return out
}
