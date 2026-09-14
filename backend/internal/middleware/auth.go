package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/training-platform/internal/model"
	"github.com/gin-gonic/gin"
)

type ctxKey string

const (
	principalKey ctxKey = "principal"
	requestIDKey ctxKey = "request_id"
)

// Principal is the authenticated caller identity derived from the session.
type Principal = model.Principal

// SetPrincipal stores the principal in context.
func SetPrincipal(c *gin.Context, p *model.Principal) {
	c.Set(string(principalKey), p)
}

// GetPrincipal returns the authenticated principal or nil.
func GetPrincipal(c *gin.Context) *model.Principal {
	v, ok := c.Get(string(principalKey))
	if !ok {
		return nil
	}
	p, _ := v.(*model.Principal)
	return p
}

// SetRequestID stores the request id in context.
func SetRequestID(c *gin.Context, id string) { c.Set(string(requestIDKey), id) }

// GetRequestID returns the request id.
func GetRequestID(c *gin.Context) string {
	v, _ := c.Get(string(requestIDKey))
	id, _ := v.(string)
	return id
}

// RequireAuth validates the session cookie and loads the principal.
func RequireAuth(sessionStore SessionLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := readSessionToken(c)
		if token == "" {
			abortUnauthorized(c)
			return
		}
		principal, err := sessionStore.LoadPrincipal(c.Request.Context(), token)
		if err != nil {
			abortUnauthorized(c)
			return
		}
		SetPrincipal(c, principal)
		c.Next()
	}
}

// RequireRole allows only the given roles; otherwise returns 403.
func RequireRole(roles ...model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetPrincipal(c)
		if p == nil || !p.IsRole(roles...) {
			abortForbidden(c, "You do not have permission to perform this action")
			return
		}
		c.Next()
	}
}

func readSessionToken(c *gin.Context) string {
	cookie, err := c.Cookie("train_session")
	if err == nil && cookie != "" {
		return cookie
	}
	// Bearer fallback for API clients/tests (never exposed client-side).
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, model.ErrorEnvelope{
		Error: model.ErrorDetail{Code: "UNAUTHENTICATED", Message: "Authentication required"},
	})
}

func abortForbidden(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorEnvelope{
		Error: model.ErrorDetail{Code: "FORBIDDEN", Message: msg},
	})
}

// SessionLoader loads a principal from a session token.
type SessionLoader interface {
	LoadPrincipal(ctx context.Context, token string) (*model.Principal, error)
}
