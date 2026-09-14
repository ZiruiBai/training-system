package database

import (
	"fmt"

	"github.com/example/training-platform/internal/config"
	"github.com/example/training-platform/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open creates a GORM instance for the environment. SQLite is used for
// development, tests and UAT; PostgreSQL is used in production.
func Open(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DBDSN)
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

// AutoMigrateSchema bootstraps the disposable SQLite schema. It must never be
// used in production.
func AutoMigrateSchema(db *gorm.DB, appEnv string) error {
	if appEnv == "production" {
		return fmt.Errorf("AutoMigrateSchema is not allowed in production")
	}
	return ensureSchema(db)
}

// EnsureSchema (re)creates the schema via GORM AutoMigrate. It is allowed in
// production when an operator explicitly enables one-time DB initialisation
// (e.g. with DB_INIT=true) so a fresh managed Postgres can be bootstrapped.
func EnsureSchema(db *gorm.DB) error {
	return ensureSchema(db)
}

func ensureSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
