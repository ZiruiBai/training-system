package handler

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/example/training-platform/internal/middleware"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/example/training-platform/internal/web"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server holds the composition root dependencies for building the router.
type Server struct {
	DB           *gorm.DB
	AuthSvc      *service.AuthService
	OrgSvc       *service.OrgService
	TrainSvc     *service.TrainingService
	Analytics    *service.AnalyticsService
	ResSvc       *service.ResourceService
	TemplateSvc  *service.TemplateService
	AgentSvc     *service.AgentService
	AgentDataSvc *service.AgentDataService
}

// NewRouter wires middleware, API routes and SPA fallback into a Gin engine.
func NewRouter(s *Server, appEnv string) *gin.Engine {
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.RequestID(), middleware.Logger())

	authMW := middleware.RequireAuth(s.AuthSvc)
	enforcer := &middleware.RBACEnforcer{}
	roleMW := enforcer.RequireRole

	// Health endpoints (no auth).
	r.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/health/ready", func(c *gin.Context) {
		sqlDB, err := s.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	api := r.Group("/api/v1")
	NewAuthHandlerWithEnv(s.AuthSvc, appEnv).RegisterRoutes(api)
	NewOrgHandler(s.OrgSvc).RegisterRoutes(api, authMW, roleMW)
	NewTrainingHandler(s.TrainSvc, s.DB).RegisterRoutes(api, authMW, roleMW)
	NewAnalyticsHandler(s.Analytics).RegisterRoutes(api, authMW, roleMW)
	NewResourceHandler(s.ResSvc).RegisterRoutes(api, authMW, roleMW)
	NewTemplateHandler(s.TemplateSvc).RegisterRoutes(api, authMW, roleMW)
	NewAgentHandler(s.AgentSvc, s.AgentDataSvc, s.DB).RegisterRoutes(api, authMW)
	NewAgentDataHandler(s.AgentDataSvc, s.DB).RegisterRoutes(api)
	NewAIHandler().RegisterRoutes(api, authMW, roleMW)
	NewCoachingHandler(service.NewCoachingService(s.DB), s.DB).RegisterRoutes(api, authMW, roleMW)

	// File uploads for training resources.
	NewUploadHandler().RegisterRoutes(api, authMW, roleMW)

	// Serve uploaded files under /uploads.
	RegisterStatic(r)

	// Register generic SPA catch-all that also handles JSON 404 for /api.
	registerStaticRoutes(r)

	return r
}

func registerStaticRoutes(r *gin.Engine) {
	dist, err := web.DistFS()
	if err != nil {
		log.Printf("web embed: %v", err)
		return
	}
	r.NoRoute(func(c *gin.Context) {
		// Unknown /api routes return JSON 404, never HTML.
		if c.Request.URL.Path == "/api" || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, model.ErrorEnvelope{
				Error: model.ErrorDetail{Code: "NOT_FOUND", Message: "API endpoint not found", RequestID: middleware.GetRequestID(c)},
			})
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusMethodNotAllowed, model.ErrorEnvelope{
				Error: model.ErrorDetail{Code: "METHOD_NOT_ALLOWED", Message: "Method not allowed"},
			})
			return
		}
		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if data, err := fs.ReadFile(dist, name); err == nil {
			httpServedFile(c, name, data)
			return
		}
		// SPA deep link: serve index.html so client-side routing works.
		if idx, err := fs.ReadFile(dist, "index.html"); err == nil {
			c.Header("Cache-Control", "no-cache")
			httpServedFile(c, "index.html", idx)
			return
		}
		notFound(c, "Page not found")
	})
}

func httpServedFile(c *gin.Context, name string, data []byte) {
	ct := contentTypeFor(name)
	c.Header("Content-Type", ct)
	// Fingerprinted assets in dist/assets/* get long-lived immutable caching.
	if strings.HasPrefix(name, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(data)
}

func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(name, ".wasm"):
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}

// RouterTestData guards the helper for tests that need the model package.
var _ = model.DashboardOverview{}
