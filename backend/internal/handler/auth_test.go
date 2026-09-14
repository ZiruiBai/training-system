package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/example/training-platform/internal/config"
	"github.com/example/training-platform/internal/database"
	"github.com/example/training-platform/internal/handler"
	"github.com/example/training-platform/internal/model"
	"github.com/example/training-platform/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Password authentication (login must always verify the bcrypt hash)
// ---------------------------------------------------------------------------

// TestLoginSuccess verifies correct username + correct password succeeds.
// Seed password for "Admin" is "System Admin123".
func TestLoginSuccess(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Admin","password":"System Admin123"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Username != "Admin" || body.Data.Role != "admin" {
		t.Fatalf("unexpected session: %+v", body.Data)
	}
}

// TestLoginWrongPassword verifies an existing account with a wrong password is
// rejected. This is the regression test for the reported privilege-escalation
// bug, where any password was accepted for an existing user.
func TestLoginWrongPassword(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Admin","password":"wrong-password"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Admin + wrong password status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	// No session cookie may be issued on a failed login.
	for _, c := range w.Result().Cookies() {
		if c.Name == "train_session" && c.Value != "" {
			t.Fatalf("failed login issued a session cookie: %s", c.Value)
		}
	}
}

// TestLoginWrongPasswordIsNotCaseInsensitive proves the password, not just the
// username, is being compared.
func TestLoginWrongPasswordCaseMismatch(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Manager","password":"Alice Manager123 "}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("password with trailing space status = %d, want 401", w.Code)
	}
}

// TestLoginUnknownUser verifies an unknown account is rejected with any password.
func TestLoginUnknownUser(t *testing.T) {
	r, _ := testServer()
	for _, pw := range []string{"anything", "System Admin123", ""} {
		w := doJSON(r, "POST", "/api/v1/auth/login",
			`{"username":"no-such-user","password":"`+pw+`"}`, "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("unknown user (password=%q) status = %d, want 401", pw, w.Code)
		}
	}
}

// TestLoginMissingPassword verifies a request without a password fails.
func TestLoginMissingPassword(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login", `{"username":"Admin"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing password status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	// Explicit empty string must behave the same way.
	w2 := doJSON(r, "POST", "/api/v1/auth/login", `{"username":"Admin","password":""}`, "")
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("empty password status = %d, want 401", w2.Code)
	}
}

// TestLoginUsernameOnlyCannotAuthenticate documents that username alone never
// produces a session.
func TestLoginUsernameOnlyCannotAuthenticate(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login", `{"username":"Admin","password":"Admin123"}`, "")
	// "Admin123" is not the seeded password ("System Admin123"); it must fail.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("near-miss password status = %d, want 401", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Session lifecycle
// ---------------------------------------------------------------------------

func TestMeUnauthenticated(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "GET", "/api/v1/auth/me", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestMeAuthenticated(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	if token == "" {
		t.Fatal("login failed")
	}
	w := doJSON(r, "GET", "/api/v1/auth/me", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// TestFailedLoginDoesNotGrantAccess proves a wrong-password attempt yields a
// token that cannot access protected endpoints.
func TestFailedLoginDoesNotGrantAccess(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Admin","password":"nope"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	// Whatever the response body contains must not be a usable session.
	body := w.Body.String()
	if len(body) > 0 {
		var probe struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &probe); err == nil && probe.Data.Token != "" {
			t.Fatalf("failed login leaked a token: %s", probe.Data.Token)
		}
	}
}

// TestProtectedAPIRequiresAuth covers the unauthenticated protected-route case.
func TestProtectedAPIRequiresAuth(t *testing.T) {
	r, _ := testServer()
	for _, path := range []string{
		"/api/v1/employees",
		"/api/v1/departments",
		"/api/v1/plans",
		"/api/v1/templates",
	} {
		w := doJSON(r, "GET", path, "", "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s without auth = %d, want 401", path, w.Code)
		}
	}
}

// TestAuthenticatedCanAccessProtectedAPI verifies the happy path still works.
func TestAuthenticatedCanAccessProtectedAPI(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	if token == "" {
		t.Fatal("login failed")
	}
	for _, path := range []string{"/api/v1/employees", "/api/v1/departments"} {
		w := doJSON(r, "GET", path, "", token)
		if w.Code != http.StatusOK {
			t.Fatalf("%s with auth = %d, want 200", path, w.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// RBAC still intact for seeded accounts
// ---------------------------------------------------------------------------

// TestRBACEmployees verifies an employee cannot create departments.
func TestRBACEmployeeCannotCreateDepartment(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Bob New Hire", "Bob New Hire123")
	if token == "" {
		t.Fatal("login failed")
	}
	w := doJSON(r, "POST", "/api/v1/departments",
		`{"name":"Hacked"}`, token)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestUnknownAPIReturnsJSON404(t *testing.T) {
	r, _ := testServer()
	w := doJSON(r, "GET", "/api/v1/nope", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Demo / passwordless login is a development-only convenience
// ---------------------------------------------------------------------------

// productionServer builds a router configured with APP_ENV=production.
func productionServer(t *testing.T) *gin.Engine {
	t.Helper()
	cfg := &config.Config{
		AppEnv: "production", DBDriver: "sqlite",
		DBDSN:      ":memory:?_pragma=foreign_keys(1)",
		SessionKey: "test-session-key-1234567890123456",
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrateSchema(db, "test"); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatal(err)
	}
	return buildRouter(db, "production")
}

// buildRouter wires a router for an arbitrary environment.
func buildRouter(db *gorm.DB, appEnv string) *gin.Engine {
	server := &handler.Server{
		DB:            db,
		AuthSvc:       service.NewAuthServiceWithEnv(db, appEnv),
		OrgSvc:        service.NewOrgService(db),
		TrainSvc:      service.NewTrainingService(db),
		Analytics:     service.NewAnalyticsService(db),
		ResSvc:        service.NewResourceService(db),
		TemplateSvc:   service.NewTemplateService(db),
		AgentSvc:      service.NewAgentService(""),
		AgentDataSvc:  service.NewAgentDataService(db),
	}
	return handler.NewRouter(server, appEnv)
}

// TestDemoLoginDisabledInProduction is the core production isolation test:
// the passwordless endpoints must not create a session in production.
func TestDemoLoginDisabledInProduction(t *testing.T) {
	r := productionServer(t)

	w := doJSON(r, "POST", "/api/v1/auth/demo-login", `{"username":"Admin"}`, "")
	if w.Code == http.StatusOK {
		t.Fatalf("SECURITY: demo-login returned 200 in production; body=%s", w.Body.String())
	}
	if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
		t.Fatalf("demo-login in production = %d, want 404 or 403", w.Code)
	}

	// The account list must not be exposed either.
	w2 := doJSON(r, "GET", "/api/v1/auth/demo-accounts", "", "")
	if w2.Code == http.StatusOK {
		t.Fatalf("SECURITY: demo-accounts returned 200 in production; body=%s", w2.Body.String())
	}
}

// TestDemoLoginInProductionCreatesNoSession proves no usable token is minted.
func TestDemoLoginInProductionCreatesNoSession(t *testing.T) {
	r := productionServer(t)
	w := doJSON(r, "POST", "/api/v1/auth/demo-login", `{"username":"Admin"}`, "")
	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200, got 200")
	}
	// Even if a token were returned it must not authenticate.
	var probe struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &probe)
	if probe.Data.Token != "" {
		t.Fatalf("SECURITY: production demo-login minted a token: %s", probe.Data.Token)
	}
	// A protected endpoint must still reject the caller.
	w2 := doJSON(r, "GET", "/api/v1/auth/me", "", probe.Data.Token)
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("me after production demo-login = %d, want 401", w2.Code)
	}
}

// TestPasswordLoginWorksInProduction ensures real password login is unaffected
// by the demo lockdown.
func TestPasswordLoginWorksInProduction(t *testing.T) {
	r := productionServer(t)
	w := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Admin","password":"System Admin123"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("production password login = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	w2 := doJSON(r, "POST", "/api/v1/auth/login",
		`{"username":"Admin","password":"bad"}`, "")
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("production wrong password = %d, want 401", w2.Code)
	}
}

// TestDemoLoginAvailableInDevelopment documents the retained dev convenience.
func TestDemoLoginAvailableInDevelopment(t *testing.T) {
	cfg := &config.Config{
		AppEnv: "development", DBDriver: "sqlite",
		DBDSN:      ":memory:?_pragma=foreign_keys(1)",
		SessionKey: "test-session-key-1234567890123456",
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrateSchema(db, "test"); err != nil {
		t.Fatal(err)
	}
	if err := database.Seed(db); err != nil {
		t.Fatal(err)
	}
	r := buildRouter(db, "development")

	if w := doJSON(r, "GET", "/api/v1/auth/demo-accounts", "", ""); w.Code != http.StatusOK {
		t.Fatalf("demo-accounts in development = %d, want 200", w.Code)
	}
	if w := doJSON(r, "POST", "/api/v1/auth/demo-login", `{"username":"Admin"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("demo-login in development = %d, want 200", w.Code)
	}
}

// TestDemoLoginAllowedFailsClosed asserts unknown environments deny demo login.
func TestDemoLoginAllowedFailsClosed(t *testing.T) {
	for _, env := range []string{"production", "Production", "PROD", "", "staging"} {
		svc := service.NewAuthServiceWithEnv(nil, env)
		if svc.DemoLoginAllowed() {
			t.Fatalf("DemoLoginAllowed() = true for env %q, want false", env)
		}
	}
	for _, env := range []string{"development", "test"} {
		svc := service.NewAuthServiceWithEnv(nil, env)
		if !svc.DemoLoginAllowed() {
			t.Fatalf("DemoLoginAllowed() = false for env %q, want true", env)
		}
	}
}

// TestSeedAccountsPreserved checks the seeded account set is intact.
func TestSeedAccountsPreserved(t *testing.T) {
	r, _ := testServer()
	token := login(r, "Admin", "System Admin123")
	if token == "" {
		t.Fatal("admin login failed")
	}
	w := doJSON(r, "GET", "/api/v1/users", "", token)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /users = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	// The response is either {"data":[...]} or {"data":{"items":[...]}}.
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	var list []model.UserSummary
	if err := json.Unmarshal(envelope.Data, &list); err != nil {
		var paged struct {
			Items []model.UserSummary `json:"items"`
		}
		if err2 := json.Unmarshal(envelope.Data, &paged); err2 != nil {
			t.Fatalf("cannot decode user list: %v / %v", err, err2)
		}
		list = paged.Items
	}
	if len(list) < 9 {
		t.Fatalf("expected at least 9 seeded users, found %d", len(list))
	}
}