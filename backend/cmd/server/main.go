package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/example/training-platform/internal/config"
	"github.com/example/training-platform/internal/database"
	"github.com/example/training-platform/internal/handler"
	"github.com/example/training-platform/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	// Development/test bootstrap via AutoMigrate (never run in production).
	// In production, an operator may opt into a one-time initialisation
	// (schema + seed) with DB_INIT=true against a fresh managed Postgres.
	if cfg.AppEnv != "production" || os.Getenv("DB_INIT") == "true" {
		if cfg.AppEnv != "production" {
			if err := database.AutoMigrateSchema(db, cfg.AppEnv); err != nil {
				log.Fatalf("migrate: %v", err)
			}
		} else {
			// Production: explicit one-time bootstrap of schema + seed data.
			if err := database.EnsureSchema(db); err != nil {
				log.Fatalf("migrate: %v", err)
			}
		}
		if err := database.Seed(db); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}

	// Auth service is bound to the runtime environment so that passwordless
	// demo login can be refused outside development/test.
	authSvc := service.NewAuthServiceWithEnv(db, cfg.AppEnv)
	orgSvc := service.NewOrgService(db)
	trainSvc := service.NewTrainingService(db)
	analyticsSvc := service.NewAnalyticsService(db)
	resSvc := service.NewResourceService(db)
	templateSvc := service.NewTemplateService(db)
	agentSvc := service.NewAgentService("")
	agentDataSvc := service.NewAgentDataService(db)

	serverDiv := &handler.Server{
		DB:           db,
		AuthSvc:      authSvc,
		OrgSvc:       orgSvc,
		TrainSvc:     trainSvc,
		Analytics:    analyticsSvc,
		ResSvc:       resSvc,
		TemplateSvc:  templateSvc,
		AgentSvc:     agentSvc,
		AgentDataSvc: agentDataSvc,
	}
	router := handler.NewRouter(serverDiv, cfg.AppEnv)

	httpServer := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("training-platform listening on :%d (env=%s, db=%s)", cfg.Port, cfg.AppEnv, cfg.DBDriver)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
