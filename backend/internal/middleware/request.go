package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/example/training-platform/internal/model"
	"github.com/gin-gonic/gin"
)

// Recovery logs panics and returns a safe internal error envelope.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic: %v", r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, model.ErrorEnvelope{
					Error: model.ErrorDetail{Code: "INTERNAL", Message: "An unexpected error occurred"},
				})
			}
		}()
		c.Next()
	}
}

// RequestID attaches or validates a request id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := ""
		if v := c.GetHeader("X-Request-Id"); len(v) <= 64 && v != "" {
			id = v
		} else {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		SetRequestID(c, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// Logger logs a structured one-line summary per request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		log.Printf("req method=%s path=%s status=%d request_id=%s latency=%s",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), GetRequestID(c), latency)
	}
}
