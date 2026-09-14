package handler

import (
	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// OrgHandler exposes departments, positions, competencies and employees.
type OrgHandler struct {
	org *service.OrgService
}

// NewOrgHandler creates an OrgHandler.
func NewOrgHandler(org *service.OrgService) *OrgHandler {
	return &OrgHandler{org: org}
}

// --- Departments ---

type departmentUpsert struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ManagerID   *uint  `json:"manager_id"`
}

func (h *OrgHandler) ListDepartments(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.org.ListDepartments(c.Request.Context(), opt)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *OrgHandler) GetDepartment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	d, err := h.org.GetDepartment(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, d)
}

func (h *OrgHandler) CreateDepartment(c *gin.Context) {
	var req departmentUpsert
	if !bindJSON(c, &req) {
		return
	}
	d, err := h.org.CreateDepartment(c.Request.Context(), req.Name, req.Description, req.ManagerID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, d)
}

func (h *OrgHandler) UpdateDepartment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req departmentUpsert
	if !bindJSON(c, &req) {
		return
	}
	d, err := h.org.UpdateDepartment(c.Request.Context(), id, req.Name, req.Description, req.ManagerID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, d)
}

func (h *OrgHandler) DeleteDepartment(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.org.DeleteDepartment(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- Positions ---

type positionUpsert struct {
	Name             string `json:"name"`
	DepartmentID     uint   `json:"department_id"`
	Description      string `json:"description"`
	Responsibilities string `json:"responsibilities"`
	Requirement      string `json:"requirement"`
	CoreAbilities    string `json:"core_abilities"`
	CycleDays        int    `json:"cycle_days"`
}

func (h *OrgHandler) ListPositions(c *gin.Context) {
	page, size := pageParams(c)
	departmentID := queryUint(c, "department_id")
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.org.ListPositions(c.Request.Context(), opt, departmentID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *OrgHandler) GetPosition(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	p, err := h.org.GetPosition(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, p)
}

func (h *OrgHandler) CreatePosition(c *gin.Context) {
	var req positionUpsert
	if !bindJSON(c, &req) {
		return
	}
	p := &model.Position{
		Name: req.Name, DepartmentID: req.DepartmentID, Description: req.Description,
		Responsibilities: req.Responsibilities, Requirement: req.Requirement,
		CoreAbilities: req.CoreAbilities, CycleDays: req.CycleDays,
	}
	p, err := h.org.CreatePosition(c.Request.Context(), p)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, p)
}

func (h *OrgHandler) UpdatePosition(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req positionUpsert
	if !bindJSON(c, &req) {
		return
	}
	p := &model.Position{
		Name: req.Name, DepartmentID: req.DepartmentID, Description: req.Description,
		Responsibilities: req.Responsibilities, Requirement: req.Requirement,
		CoreAbilities: req.CoreAbilities, CycleDays: req.CycleDays,
	}
	p, err := h.org.UpdatePosition(c.Request.Context(), id, p)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, p)
}

func (h *OrgHandler) DeletePosition(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.org.DeletePosition(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- Competencies ---

func (h *OrgHandler) ListCompetencies(c *gin.Context) {
	positionID, parsed := parseID(c)
	if !parsed {
		return
	}
	items, err := h.org.ListCompetencies(c.Request.Context(), positionID)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, items)
}

func (h *OrgHandler) UpsertCompetencies(c *gin.Context) {
	positionID, parsed := parseID(c)
	if !parsed {
		return
	}
	var items []model.PositionCompetency
	if !bindJSON(c, &items) {
		return
	}
	if err := h.org.UpsertCompetencies(c.Request.Context(), positionID, items); err != nil {
		mapServiceError(c, err)
		return
	}
	list, _ := h.org.ListCompetencies(c.Request.Context(), positionID)
	ok(c, list)
}

// --- Employees ---

type employeeUpsert struct {
	EmployeeCode   string               `json:"employee_code"`
	Name           string               `json:"name"`
	Email          string               `json:"email"`
	Phone          string               `json:"phone"`
	DepartmentID   uint                 `json:"department_id"`
	PositionID     uint                 `json:"position_id"`
	ManagerID      *uint                `json:"manager_id"`
	HireDate       *timeFromJSON        `json:"hire_date"`
	WorkExperience int                  `json:"work_experience"`
	SkillLevel     model.SkillLevel     `json:"skill_level"`
	CurrentStage   int                  `json:"current_stage"`
	TrainingStatus model.TrainingStatus `json:"training_status"`
	Progress       float64              `json:"progress"`
}

func (h *OrgHandler) ListEmployees(c *gin.Context) {
	page, size := pageParams(c)
	departmentID := queryUint(c, "department_id")
	positionID := queryUint(c, "position_id")
	status := c.Query("status")
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	var items []model.Employee
	var total int64
	var err error
	if p := middleware.GetPrincipal(c); p.IsRole(model.RoleManager) {
		items, total, err = h.org.ListEmployeesByManager(c.Request.Context(), p.UserID, opt)
	} else {
		items, total, err = h.org.ListEmployees(c.Request.Context(), opt, departmentID, positionID, status)
	}
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *OrgHandler) GetEmployee(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	e, err := h.org.GetEmployee(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	// Managers can only view employees in their own department (or direct reports).
	if p := middleware.GetPrincipal(c); p.IsRole(model.RoleManager) {
		if !h.org.ManagerCanAccessEmployee(c.Request.Context(), p.UserID, e) {
			fail403(c)
			return
		}
	}
	ok(c, e)
}

func (h *OrgHandler) CreateEmployee(c *gin.Context) {
	var req employeeUpsert
	if !bindJSON(c, &req) {
		return
	}
	e := modelFromEmployeeUpsert(req)
	e, err := h.org.CreateEmployee(c.Request.Context(), e)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, e)
}

func (h *OrgHandler) UpdateEmployee(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var req employeeUpsert
	if !bindJSON(c, &req) {
		return
	}
	e := modelFromEmployeeUpsert(req)
	e, err := h.org.UpdateEmployee(c.Request.Context(), id, e)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, e)
}

func (h *OrgHandler) DeleteEmployee(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.org.DeleteEmployee(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// RegisterRoutes wires org endpoints.
func (h *OrgHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	departments := r.Group("/departments", auth, role(model.RoleAdmin))
	{
		departments.GET("", h.ListDepartments)
		departments.GET("/:id", h.GetDepartment)
		departments.POST("", h.CreateDepartment)
		departments.PUT("/:id", h.UpdateDepartment)
		departments.DELETE("/:id", h.DeleteDepartment)
	}
	positions := r.Group("/positions", auth, role(model.RoleAdmin))
	{
		positions.GET("", h.ListPositions)
		positions.GET("/:id", h.GetPosition)
		positions.POST("", h.CreatePosition)
		positions.PUT("/:id", h.UpdatePosition)
		positions.DELETE("/:id", h.DeletePosition)
		positions.GET("/:id/competencies", h.ListCompetencies)
		positions.PUT("/:id/competencies", h.UpsertCompetencies)
	}
	employees := r.Group("/employees", auth)
	{
		employees.GET("", h.ListEmployees)
		employees.GET("/:id", h.GetEmployee)
		employees.POST("", role(model.RoleAdmin), h.CreateEmployee)
		employees.PUT("/:id", role(model.RoleAdmin), h.UpdateEmployee)
		employees.DELETE("/:id", role(model.RoleAdmin), h.DeleteEmployee)
	}
}
