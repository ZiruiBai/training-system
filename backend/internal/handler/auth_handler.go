package handler

import (
	"errors"
	"net/http"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler exposes login, session and logout endpoints.
type AuthHandler struct {
	auth   *service.AuthService
	appEnv string
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler { return &AuthHandler{auth: auth} }

// NewAuthHandlerWithEnv creates an AuthHandler bound to a runtime environment.
// Passwordless demo endpoints are only registered outside production.
func NewAuthHandlerWithEnv(auth *service.AuthService, appEnv string) *AuthHandler {
	return &AuthHandler{auth: auth, appEnv: appEnv}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	// Password is mandatory: an empty value must fail authentication rather
	// than fall back to a passwordless path. Validation happens in the service
	// layer so that "missing" and "wrong" produce the same 401 response.
	Password string `json:"password"`
}

// Login authenticates and issues a session cookie.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindJSON(c, &req) {
		return
	}
	token, info, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password", nil)
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			fail(c, http.StatusForbidden, "ACCOUNT_INACTIVE", "This account is inactive", nil)
			return
		}
		internal(c)
		return
	}
	secure := c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("train_session", token, int(24*60*60), "/", "", secure, true)
	ok(c, info)
}

// Me returns the current session identity.
func (h *AuthHandler) Me(c *gin.Context) {
	p := middleware.GetPrincipal(c)
	if p == nil {
		fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required", nil)
		return
	}
	info := &model.SessionInfo{
		UserID: p.UserID, Username: p.Username, Role: p.Role,
		Display: p.DisplayName, EmployeeID: p.EmployeeID,
	}
	// Load email for display.
	ok(c, info)
}

// Logout clears the session.
func (h *AuthHandler) Logout(c *gin.Context) {
	p := middleware.GetPrincipal(c)
	if p != nil {
		if token, err := c.Cookie("train_session"); err == nil {
			_ = h.auth.Logout(c.Request.Context(), token, p.UserID)
		}
	}
	c.SetCookie("train_session", "", -1, "/", "", false, true)
	ok(c, gin.H{"status": "logged_out"})
}

// DemoAccounts returns the one-click login account list.
// In production these routes are not registered at all; this handler keeps a
// defence-in-depth check in case the wiring changes.
func (h *AuthHandler) DemoAccounts(c *gin.Context) {
	if !h.auth.DemoLoginAllowed() {
		fail(c, http.StatusNotFound, "NOT_FOUND", "Not found", nil)
		return
	}
	accts, err := h.auth.DemoAccounts(c.Request.Context())
	if err != nil {
		internal(c)
		return
	}
	ok(c, accts)
}

// DemoLogin signs in a demo account without a password (non-production only).
// In production it is disabled and creates no session.
func (h *AuthHandler) DemoLogin(c *gin.Context) {
	if !h.auth.DemoLoginAllowed() {
		fail(c, http.StatusNotFound, "NOT_FOUND", "Not found", nil)
		return
	}
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if !bindJSON(c, &req) {
		return
	}
	token, info, err := h.auth.DemoLogin(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, service.ErrDemoLoginDisabled) {
			fail(c, http.StatusForbidden, "DEMO_LOGIN_DISABLED", "Demo login is not available", nil)
			return
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "INVALID_ACCOUNT", "Unknown demo account", nil)
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			fail(c, http.StatusForbidden, "ACCOUNT_INACTIVE", "This account is inactive", nil)
			return
		}
		internal(c)
		return
	}
	secure := c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie("train_session", token, int(24*60*60), "/", "", secure, true)
	ok(c, info)
}

// RegisterRoutes wires auth endpoints.
//
// Password-authenticated endpoints are always available. Passwordless demo
// endpoints are registered ONLY when demo login is permitted, so in production
// they do not exist and return 404 even before reaching a handler.
func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", middleware.RequireAuth(h.auth), h.Logout)
		auth.GET("/me", middleware.RequireAuth(h.auth), h.Me)
		if h.auth.DemoLoginAllowed() {
			auth.GET("/demo-accounts", h.DemoAccounts)
			auth.POST("/demo-login", h.DemoLogin)
		}
	}
}
