package handler

import (
	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// ResourceHandler exposes courses, materials, SOPs and users.
type ResourceHandler struct {
	res *service.ResourceService
}

// NewResourceHandler creates a ResourceHandler.
func NewResourceHandler(res *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{res: res}
}

func (h *ResourceHandler) ListCourses(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.res.ListCourses(c.Request.Context(), opt, c.Query("category"), c.Query("status"))
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *ResourceHandler) GetCourse(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	v, err := h.res.GetCourse(c.Request.Context(), id)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, v)
}

func (h *ResourceHandler) CreateCourse(c *gin.Context) {
	var body model.Course
	if !bindJSON(c, &body) {
		return
	}
	v, err := h.res.CreateCourse(c.Request.Context(), &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, v)
}

func (h *ResourceHandler) UpdateCourse(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var body model.Course
	if !bindJSON(c, &body) {
		return
	}
	v, err := h.res.UpdateCourse(c.Request.Context(), id, &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, v)
}

func (h *ResourceHandler) DeleteCourse(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.res.DeleteCourse(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- Materials ---

func (h *ResourceHandler) ListMaterials(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.res.ListMaterials(c.Request.Context(), opt, c.Query("category"))
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *ResourceHandler) CreateMaterial(c *gin.Context) {
	var body model.TrainingMaterial
	if !bindJSON(c, &body) {
		return
	}
	if p := middleware.GetPrincipal(c); p != nil {
		body.UploaderID = &p.UserID
	}
	v, err := h.res.CreateMaterial(c.Request.Context(), &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, v)
}

func (h *ResourceHandler) DeleteMaterial(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.res.DeleteMaterial(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- SOPs ---

func (h *ResourceHandler) ListSops(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.res.ListSops(c.Request.Context(), opt, c.Query("sop_type"))
	if err != nil {
		mapServiceError(c, err)
		return
	}
	list(c, items, page, size, total)
}

func (h *ResourceHandler) CreateSop(c *gin.Context) {
	var body model.SopDocument
	if !bindJSON(c, &body) {
		return
	}
	v, err := h.res.CreateSop(c.Request.Context(), &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	created(c, v)
}

func (h *ResourceHandler) UpdateSop(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var body model.SopDocument
	if !bindJSON(c, &body) {
		return
	}
	v, err := h.res.UpdateSop(c.Request.Context(), id, &body)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, v)
}

func (h *ResourceHandler) DeleteSop(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.res.DeleteSop(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// --- Users (admin) ---

func (h *ResourceHandler) ListUsers(c *gin.Context) {
	page, size := pageParams(c)
	opt := service.ListOptions{Page: page, PageSize: size, Search: c.Query("search")}
	items, total, err := h.res.ListUsers(c.Request.Context(), opt)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	// Never leak password hashes.
	for i := range items {
		items[i].Password = ""
	}
	list(c, items, page, size, total)
}

func (h *ResourceHandler) CreateUser(c *gin.Context) {
	var body struct {
		Username    string     `json:"username"`
		Email       string     `json:"email"`
		Password    string     `json:"password"`
		DisplayName string     `json:"display_name"`
		Role        model.Role `json:"role"`
		EmployeeID  *uint      `json:"employee_id"`
	}
	if !bindJSON(c, &body) {
		return
	}
	u := &model.User{Username: body.Username, Email: body.Email, DisplayName: body.DisplayName,
		Role: body.Role, EmployeeID: body.EmployeeID, Active: true}
	u, err := h.res.CreateUser(c.Request.Context(), u, body.Password)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	u.Password = ""
	created(c, u)
}

func (h *ResourceHandler) UpdateUser(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	var body struct {
		Email       string     `json:"email"`
		DisplayName string     `json:"display_name"`
		Role        model.Role `json:"role"`
		Active      bool       `json:"active"`
		EmployeeID  *uint      `json:"employee_id"`
		Password    string     `json:"password"`
	}
	if !bindJSON(c, &body) {
		return
	}
	u := &model.User{Email: body.Email, DisplayName: body.DisplayName, Role: body.Role,
		Active: body.Active, EmployeeID: body.EmployeeID}
	u, err := h.res.UpdateUser(c.Request.Context(), id, u, body.Password)
	if err != nil {
		mapServiceError(c, err)
		return
	}
	u.Password = ""
	ok(c, u)
}

func (h *ResourceHandler) DeleteUser(c *gin.Context) {
	id, parsed := parseID(c)
	if !parsed {
		return
	}
	if err := h.res.DeleteUser(c.Request.Context(), id); err != nil {
		mapServiceError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

// RegisterRoutes wires resource endpoints.
func (h *ResourceHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, role middleware.RoleFunc) {
	courses := r.Group("/courses", auth)
	{
		courses.GET("", h.ListCourses)
		courses.GET("/:id", h.GetCourse)
		courses.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateCourse)
		courses.PUT("/:id", role(model.RoleManager, model.RoleAdmin), h.UpdateCourse)
		courses.DELETE("/:id", role(model.RoleManager, model.RoleAdmin), h.DeleteCourse)
	}
	materials := r.Group("/materials", auth)
	{
		materials.GET("", h.ListMaterials)
		materials.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateMaterial)
		materials.DELETE("/:id", role(model.RoleManager, model.RoleAdmin), h.DeleteMaterial)
	}
	sops := r.Group("/sops", auth)
	{
		sops.GET("", h.ListSops)
		sops.POST("", role(model.RoleManager, model.RoleAdmin), h.CreateSop)
		sops.PUT("/:id", role(model.RoleManager, model.RoleAdmin), h.UpdateSop)
		sops.DELETE("/:id", role(model.RoleManager, model.RoleAdmin), h.DeleteSop)
	}
	users := r.Group("/users", auth, role(model.RoleAdmin))
	{
		users.GET("", h.ListUsers)
		users.POST("", h.CreateUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeleteUser)
	}
}
