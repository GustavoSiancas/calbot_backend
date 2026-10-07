package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"calbot/db"
	"calbot/internal/config"
	"calbot/internal/feedback"
	"calbot/internal/httpapi"
	"calbot/internal/menu"
	"calbot/internal/postgres"
	"calbot/internal/security"
	"calbot/internal/user"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	databaseURL := requiredEnv("DATABASE_URL")
	jwtSecret := requiredEnv("JWT_SECRET")
	jwtTTL := durationEnv("JWT_TTL", 24*time.Hour)

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	if err := db.ApplyInitialSchema(context.Background(), pool); err != nil {
		log.Fatalf("apply database schema: %v", err)
	}

	jwtService, err := security.NewJWTService(jwtSecret, jwtTTL)
	if err != nil {
		log.Fatalf("configure JWT service: %v", err)
	}
	authService := user.NewService(postgres.NewUserStore(pool), jwtService)
	authHandler := httpapi.NewAuthHandler(authService)
	publicMenuService := menu.NewService(postgres.NewMenuOptionStore(pool), jwtService)
	publicMenuHandler := httpapi.NewPublicMenuHandler(publicMenuService)
	staffMenuHandler := httpapi.NewStaffMenuHandler(publicMenuService)
	staffPromptHandler := httpapi.NewStaffPromptHandler(publicMenuService)
	staffResponseGroupHandler := httpapi.NewStaffResponseGroupHandler(publicMenuService)
	publicFeedbackService := feedback.NewService(postgres.NewContentFeedbackStore(pool))
	publicFeedbackHandler := httpapi.NewPublicFeedbackHandler(publicFeedbackService)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	publicMenuHandler.RegisterRoutes(mux)
	staffMenuHandler.RegisterRoutes(mux)
	staffPromptHandler.RegisterRoutes(mux)
	staffResponseGroupHandler.RegisterRoutes(mux)
	publicFeedbackHandler.RegisterRoutes(mux)
	httpapi.RegisterSwagger(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	address := envOrDefault("HTTP_ADDR", ":8080")
	server := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("API listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("%s must be a valid duration: %v", name, err)
	}
	return duration
}
