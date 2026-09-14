package middleware

import (
	"net/http"

	"github.com/example/training-platform/internal/model"
	"github.com/gin-gonic/gin"
)

// RoleFunc returns an authorization middleware that allows only the given roles.
type RoleFunc func(roles ...model.Role) gin.HandlerFunc

// RBACEnforcer wraps role authorization.
type RBACEnforcer struct{}

// RequireRole builds an authorization middleware for specific roles.
func (e *RBACEnforcer) RequireRole(roles ...model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetPrincipal(c)
		if p == nil || !p.IsRole(roles...) {
			c.AbortWithStatusJSON(http.StatusForbidden, model.ErrorEnvelope{
				Error: model.ErrorDetail{Code: "FORBIDDEN", Message: "You do not have permission to perform this action"},
			})
			return
		}
		c.Next()
	}
}
