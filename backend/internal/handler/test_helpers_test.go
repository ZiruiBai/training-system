package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/example/training-platform/internal/config"
	"github.com/example/training-platform/internal/database"
	"github.com/example/training-platform/internal/handler"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// testServer builds an in-memory router with a test database and seeded data.
func testServer() (*gin.Engine, *gorm.DB) {
	cfg := &config.Config{
		AppEnv: "test", DBDriver: "sqlite",
		DBDSN:      ":memory:?_pragma=foreign_keys(1)",
		SessionKey: "test-session-key-1234567890123456",
	}
	db, err := database.Open(cfg)
	if err != nil {
		panic(err)
	}
	if err := database.AutoMigrateSchema(db, "test"); err != nil {
		panic(err)
	}
	if err := database.Seed(db); err != nil {
		panic(err)
	}
	// Bind the auth service to the test environment so demo login gating can
	// be exercised explicitly in tests.
	authSvc := service.NewAuthServiceWithEnv(db, "test")
	orgSvc := service.NewOrgService(db)
	trainSvc := service.NewTrainingService(db)
	analyticsSvc := service.NewAnalyticsService(db)
	resSvc := service.NewResourceService(db)
	templateSvc := service.NewTemplateService(db)
	agentSvc := service.NewAgentService("")
	agentDataSvc := service.NewAgentDataService(db)
	server := &handler.Server{
		DB:            db,
		AuthSvc:       authSvc,
		OrgSvc:        orgSvc,
		TrainSvc:      trainSvc,
		Analytics:     analyticsSvc,
		ResSvc:        resSvc,
		TemplateSvc:   templateSvc,
		AgentSvc:      agentSvc,
		AgentDataSvc:  agentDataSvc,
	}
	return handler.NewRouter(server, "test"), db
}

func doJSON(r *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("{}")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// login returns a session token for a seeded user by parsing the cookie.
func login(r *gin.Engine, username, password string) string {
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"`+username+`","password":"`+password+`"}`, "")
	if w.Code != http.StatusOK {
		return ""
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "train_session" {
			return c.Value
		}
	}
	return ""
}

var _ = json.Marshal
var _ = model.DashboardOverview{}
