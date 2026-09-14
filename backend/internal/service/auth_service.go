package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/example/training-platform/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	// ErrInvalidCredentials is returned when login details are wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserInactive is returned when the account is disabled.
	ErrUserInactive = errors.New("account is inactive")
	// ErrDemoLoginDisabled is returned when demo login is attempted in an
	// environment where it is not allowed (e.g. production).
	ErrDemoLoginDisabled = errors.New("demo login is disabled in this environment")
)

// AuthService handles login and session lifecycle.
type AuthService struct {
	db *gorm.DB
	// appEnv is the runtime environment ("development", "test", "production").
	// Passwordless demo login is only permitted outside production.
	appEnv string
}

// NewAuthService creates an AuthService.
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

// NewAuthServiceWithEnv creates an AuthService bound to a runtime environment.
// Password verification is always enforced; appEnv only gates demo login.
func NewAuthServiceWithEnv(db *gorm.DB, appEnv string) *AuthService {
	return &AuthService{db: db, appEnv: appEnv}
}

// Environment returns the configured runtime environment.
func (s *AuthService) Environment() string { return s.appEnv }

// DemoLoginAllowed reports whether passwordless demo login may be used.
// It is disabled in production and whenever the environment is unknown.
func (s *AuthService) DemoLoginAllowed() bool {
	switch strings.ToLower(strings.TrimSpace(s.appEnv)) {
	case "development", "dev", "test", "testing", "local":
		return true
	default:
		// production and every unrecognised value fail closed.
		return false
	}
}

// Login validates credentials and creates a session.
//
// A session is issued only when the account exists, is active AND the supplied
// password matches the stored bcrypt hash. There is no code path that issues a
// session based on username alone.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.SessionInfo, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", nil, ErrInvalidCredentials
	}

	var user model.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Run a dummy comparison so that response timing does not reveal
			// whether the account exists.
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if !user.Active {
		return "", nil, ErrUserInactive
	}

	// Mandatory password verification against the stored bcrypt hash.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	return s.issueSession(ctx, &user)
}

// issueSession generates a token and persists the session row.
func (s *AuthService) issueSession(ctx context.Context, user *model.User) (string, *model.SessionInfo, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", nil, err
	}
	session := model.Session{Token: token, UserID: user.ID, ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour)}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return "", nil, err
	}
	info := &model.SessionInfo{
		UserID: user.ID, Username: user.Username, Role: user.Role,
		Display: user.DisplayName, Email: user.Email, EmployeeID: user.EmployeeID,
	}
	return token, info, nil
}

// LoadPrincipal resolves a session token to a principal.
func (s *AuthService) LoadPrincipal(ctx context.Context, token string) (*model.Principal, error) {
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var session model.Session
	if err := s.db.WithContext(ctx).Where("token = ? AND expires_at > ?", token, time.Now().UTC()).First(&session).Error; err != nil {
		return nil, err
	}
	var user model.User
	if err := s.db.WithContext(ctx).Where("id = ? AND active = ?", session.UserID, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &model.Principal{
		UserID: user.ID, Username: user.Username, Role: user.Role,
		EmployeeID: user.EmployeeID, DisplayName: user.DisplayName,
	}, nil
}

// Logout deletes the session belonging to the principal.
func (s *AuthService) Logout(ctx context.Context, token string, userID uint) error {
	return s.db.WithContext(ctx).Where("token = ? AND user_id = ?", token, userID).Delete(&model.Session{}).Error
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// dummyHash is a valid bcrypt hash used to equalise timing for unknown users.
var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

// DemoAccount is a one-click login option for the MVP demo screen.
type DemoAccount struct {
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	Role         model.Role `json:"role"`
	EmployeeID   *uint      `json:"employee_id"`
	EmployeeCode string     `json:"employee_code,omitempty"`
}

// DemoAccounts returns the accounts shown on the one-click login screen.
// It is only reachable when DemoLoginAllowed() is true.
func (s *AuthService) DemoAccounts(ctx context.Context) ([]DemoAccount, error) {
	// Ordered: admin, manager, then employee users with codes for the 3 new hires.
	names := []string{"Admin", "Manager", "employee", "Dev", "Sarah", "Tom"}
	var users []model.User
	if err := s.db.WithContext(ctx).Where("username IN ?", names).Find(&users).Error; err != nil {
		return nil, err
	}
	byUsername := map[string]model.User{}
	for _, u := range users {
		byUsername[u.Username] = u
	}
	order := names
	out := make([]DemoAccount, 0, len(order))
	for _, n := range order {
		u, ok := byUsername[n]
		if !ok {
			continue
		}
		acct := DemoAccount{
			Username: u.Username, DisplayName: u.DisplayName, Role: u.Role, EmployeeID: u.EmployeeID,
		}
		if u.EmployeeID != nil {
			var emp model.Employee
			if err := s.db.WithContext(ctx).First(&emp, *u.EmployeeID).Error; err == nil {
				acct.EmployeeCode = emp.EmployeeCode
			}
		}
		out = append(out, acct)
	}
	return out, nil
}

// DemoLogin signs in a demo account by username alone.
//
// SECURITY: this is a passwordless path and is therefore refused unless the
// service is explicitly running in a non-production environment. Callers must
// not bypass this guard. In production the request fails with
// ErrDemoLoginDisabled and no session is created.
func (s *AuthService) DemoLogin(ctx context.Context, username string) (string, *model.SessionInfo, error) {
	if !s.DemoLoginAllowed() {
		return "", nil, ErrDemoLoginDisabled
	}
	var user model.User
	if err := s.db.WithContext(ctx).Where("username = ?", strings.TrimSpace(username)).First(&user).Error; err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if !user.Active {
		return "", nil, ErrUserInactive
	}
	return s.issueSession(ctx, &user)
}