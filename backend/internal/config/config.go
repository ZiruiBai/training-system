package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds validated server configuration values.
type Config struct {
	AppEnv      string
	Port        int
	DBDriver    string
	DBDSN       string
	SessionKey  string
	SessionDays int
}

// Load reads configuration from the environment with sensible local defaults.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:      envOr("APP_ENV", "development"),
		Port:        envIntOr("PORT", 8080),
		DBDriver:    envOr("DB_DRIVER", "sqlite"),
		DBDSN:       dbDSN(),
		SessionKey:  envOr("SESSION_KEY", "dev-session-key-change-me"),
		SessionDays: envIntOr("SESSION_DAYS", 7),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// dbDSN resolves the DB DSN. For Postgres, prefer the platform-injected
// DATABASE_URL when DB_DSN is not set; otherwise fall back to DB_DSN or a
// local SQLite path.
func dbDSN() string {
	if v := os.Getenv("DB_DSN"); v != "" {
		return v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" && os.Getenv("DB_DRIVER") == "postgres" {
		return v
	}
	return "var/training.db?_pragma=foreign_keys(1)"
}

func (c *Config) validate() error {
	switch c.AppEnv {
	case "development", "test", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of development, test, production; got %q", c.AppEnv)
	}
	if c.DBDriver == "" {
		return fmt.Errorf("DB_DRIVER is required")
	}
	if c.AppEnv == "production" && c.DBDriver != "postgres" {
		return fmt.Errorf("production requires DB_DRIVER=postgres, got %q", c.DBDriver)
	}
	if len(c.SessionKey) < 16 {
		return fmt.Errorf("SESSION_KEY must be at least 16 characters in length")
	}
	return nil
}

// SessionTTL returns the session cookie lifetime.
func (c *Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionDays) * 24 * time.Hour
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
