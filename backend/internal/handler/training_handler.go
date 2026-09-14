package handler

import (
	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TrainingHandler exposes plans, stages, tasks and learning records.
type TrainingHandler struct {
	training *service.TrainingService
	db       *gorm.DB
}

// NewTrainingHandler creates a TrainingHandler.
func NewTrainingHandler(training *service.TrainingService, db *gorm.DB) *TrainingHandler {
	return &TrainingHandler{training: training, db: db}
}

func (h *TrainingHandler) ListPlans(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	departmentID := queryUint(c, "department_id")
	status := c.Query("status")
	// Employees can only list their own plans.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleEmployee) && p.EmployeeID != nil {
		employeeID = *p.EmployeeID
	}
	// Managers can only list plans of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		ids, err := service.ManagedEmployeeIDs(c.Request.Context(), h.db, p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		opt.EmployeeIDs = ids
	}
	items, total, err := h.training.ListPlans(c.Request.Context(), opt, employeeID, departmentID, status)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *TrainingHandler) GetPlan(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	plan, err := h.training.GetPlan(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if err := h.canAccessEmployeePlan(c, plan.EmployeeID); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, plan)
}

type planUpsert struct {
	EmployeeID   uint             `json:"employee_id"`
	StartDate    *timeFromJSON    `json:"start_date"`
	EndDate      *timeFromJSON    `json:"end_date"`
	CycleDays    int              `json:"cycle_days"`
	CurrentStage int              `json:"current_stage"`
	Status       model.PlanStatus `json:"status"`
}

func (h *TrainingHandler) CreatePlan(c *gin.Context) {
	var req planUpsert
	if !bindJSON(c, &req) {
		return
	}
	plan := &model.TrainingPlan{EmployeeID: req.EmployeeID, CycleDays: req.CycleDays}
	if req.StartDate != nil {
		plan.StartDate = req.StartDate.Time()
	}
	if req.EndDate != nil {
		plan.EndDate = req.EndDate.Time()
	}
	p := middleware.GetPrincipal(c)
	var createdBy *uint
	if p != nil {
		createdBy = &p.UserID
	}
	plan, err := h.training.CreatePlanWithStages(c.Request.Context(), plan, createdBy, model.SourceManual)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, plan)
}

func (h *TrainingHandler) UpdatePlan(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req planUpsert
	if !bindJSON(c, &req) {
		return
	}
	plan := &model.TrainingPlan{EmployeeID: req.EmployeeID, CycleDays: req.CycleDays, CurrentStage: req.CurrentStage}
	if req.StartDate != nil {
		plan.StartDate = req.StartDate.Time()
	}
	if req.EndDate != nil {
		plan.EndDate = req.EndDate.Time()
	}
	plan, err := h.training.UpdatePlan(c.Request.Context(), id, plan)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, plan)
}

func (h *TrainingHandler) TransitionPlan(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req struct {
		Status model.PlanStatus `json:"status"`
	}
	if !bindJSON(c, &req) {
		return
	}
	p := middleware.GetPrincipal(c)
	if p == nil {
		fail403(c)
		return
	}
	plan, err := h.training.Transition(c.Request.Context(), id, req.Status, p)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, plan)
}

func (h *TrainingHandler) canAccessEmployeePlan(c *gin.Context, employeeID uint) error {
	p := middleware.GetPrincipal(c)
	if p == nil {
		return service.ErrForbidden
	}
	if p.IsRole(model.RoleAdmin) {
		return nil
	}
	if p.IsRole(model.RoleEmployee) {
		if p.EmployeeID != nil && *p.EmployeeID == employeeID {
			return nil
		}
		return service.ErrForbidden
	}
	// Manager: allow if employee is a direct report or in a department they manage.
	if p.Role == model.RoleManager {
		var e model.Employee
		if err := h.db.Where("id = ?", employeeID).First(&e).Error; err == nil {
			if service.ManagerCanAccessEmployee(c.Request.Context(), h.db, p.UserID, &e) {
				return nil
			}
		}
		return service.ErrForbidden
	}
	return service.ErrForbidden
}

// --- Stages ---

func (h *TrainingHandler) ListStages(c *gin.Context) {
	planID := queryUint(c, "plan_id")
	items, err := h.training.ListStages(c.Request.Context(), planID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, items)
}

// --- Tasks ---

func (h *TrainingHandler) ListTasks(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	planID := queryUint(c, "plan_id")
	stageID := queryUint(c, "stage_id")
	status := c.Query("status")
	// Employees can only see their own tasks.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleEmployee) && p.EmployeeID != nil {
		employeeID = *p.EmployeeID
	}
	items, total, err := h.training.ListTasks(c.Request.Context(), opt, employeeID, planID, stageID, status)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *TrainingHandler) GetTask(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	task, err := h.training.GetTask(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	if p := middleware.GetPrincipal(c); p != nil {
		if p.IsRole(model.RoleEmployee) {
			if p.EmployeeID == nil || *p.EmployeeID != task.EmployeeID {
				fail403(c)
				return
			}
		}
	}
	ok(c, task)
}

type taskUpsert struct {
	EmployeeID    uint             `json:"employee_id"`
	PlanID        uint             `json:"plan_id"`
	StageID       uint             `json:"stage_id"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	TaskType      model.TaskType   `json:"task_type"`
	CourseID      *uint            `json:"course_id"`
	MaterialID    *uint            `json:"material_id"`
	StartDate     *timeFromJSON    `json:"start_date"`
	DueDate       *timeFromJSON    `json:"due_date"`
	EstimateHours float64          `json:"estimate_hours"`
	Priority      model.Priority   `json:"priority"`
	Status        model.TaskStatus `json:"status"`
	Outcome       string           `json:"outcome"`
	Score         *float64         `json:"score"`
}

func modelFromTaskUpsert(req taskUpsert) *model.LearningTask {
	t := &model.LearningTask{
		EmployeeID: req.EmployeeID, PlanID: req.PlanID, StageID: req.StageID,
		Title: req.Title, Description: req.Description, TaskType: req.TaskType,
		CourseID: req.CourseID, MaterialID: req.MaterialID, EstimateHours: req.EstimateHours,
		Priority: req.Priority, Status: req.Status, Outcome: req.Outcome, Score: req.Score,
	}
	if req.StartDate != nil {
		t.StartDate = req.StartDate.Time()
	}
	if req.DueDate != nil {
		t.DueDate = req.DueDate.Time()
	}
	return t
}

func (h *TrainingHandler) CreateTask(c *gin.Context) {
	var req taskUpsert
	if !bindJSON(c, &req) {
		return
	}
	task := modelFromTaskUpsert(req)
	task, err := h.training.CreateTask(c.Request.Context(), task)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, task)
}

func (h *TrainingHandler) UpdateTask(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req taskUpsert
	if !bindJSON(c, &req) {
		return
	}
	task := modelFromTaskUpsert(req)
	task, err := h.training.UpdateTask(c.Request.Context(), id, task)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, task)
}

func (h *TrainingHandler) CompleteTask(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	task, err := h.training.GetTask(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	p := middleware.GetPrincipal(c)
	if p != nil && p.IsRole(model.RoleEmployee) {
		if p.EmployeeID == nil || *p.EmployeeID != task.EmployeeID {
			fail403(c)
			return
		}
	}
	var req struct {
		Outcome string   `json:"outcome"`
		Score   *float64 `json:"score"`
		Answers []int    `json:"answers"`
	}
	if !bindJSON(c, &req) {
		return
	}
	task, err = h.training.CompleteTask(c.Request.Context(), id, req.Outcome, req.Score, req.Answers)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, task)
}

func (h *TrainingHandler) DeleteTask(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.training.DeleteTask(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- Learning records ---

func (h *TrainingHandler) ListRecords(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	employeeID := queryUint(c, "employee_id")
	taskID := queryUint(c, "task_id")
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleEmployee) && p.EmployeeID != nil {
		employeeID = *p.EmployeeID
	}
	// Managers can only list records of employees in their own department.
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleManager) {
		ids, err := service.ManagedEmployeeIDs(c.Request.Context(), h.db, p.UserID)
		if err != nil {
			mapServiceError(c, err)
			return
		}
		opt.EmployeeIDs = ids
	}
	items, total, err := h.training.ListRecords(c.Request.Context(), opt, employeeID, taskID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *TrainingHandler) CreateRecord(c *gin.Context) {
	var req model.LearningRecord
	if !bindJSON(c, &req) {
		return
	}
	rec, err := h.training.CreateRecord(c.Request.Context(), &req)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, rec)
}

func (h *TrainingHandler) Stats(c *gin.Context) {
	employeeID := queryUint(c, "employee_id")
	if p := middleware.GetPrincipal(c); p != nil && p.IsRole(model.RoleEmployee) && p.EmployeeID != nil {
		employeeID = *p.EmployeeID
	}
	if employeeID == 0 {
		badRequest(c, "employee_id is required")
		return
	}
	stats, err := h.training.LearningStats(c.Request.Context(), employeeID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, stats)
}

// RegisterRoutes wires training endpoints.
func (h *TrainingHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	plans := r.Group("/plans", auth)
	{
		plans.GET("", h.ListPlans)
		plans.GET("/:id", h.GetPlan)
		plans.POST("", role(model.RoleAdmin), h.CreatePlan)
		plans.PUT("/:id", role(model.RoleAdmin), h.UpdatePlan)
		plans.POST("/:id/transition", role(model.RoleManager, model.RoleAdmin), h.TransitionPlan)
	}
	stages := r.Group("/stages", auth)
	{
		stages.GET("", h.ListStages)
	}
	tasks := r.Group("/tasks", auth)
	{
		tasks.GET("", h.ListTasks)
		tasks.GET("/:id", h.GetTask)
		tasks.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateTask)
		tasks.PUT("/:id", role(model.RoleManager, model.RoleAdmin), h.UpdateTask)
		tasks.POST("/:id/complete", h.CompleteTask)
		tasks.DELETE("/:id", role(model.RoleAdmin), h.DeleteTask)
	}
	records := r.Group("/records", auth)
	{
		records.GET("", h.ListRecords)
		records.POST("", h.CreateRecord)
	}
	r.GET("/stats/learning", auth, h.Stats)
}
