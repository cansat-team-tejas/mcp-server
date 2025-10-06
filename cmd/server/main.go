package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"

	"goapp/internal/ai"
	"goapp/internal/api"
	"goapp/internal/config"
	"goapp/internal/telemetry"
)

func main() {
	cfg := config.Load()

	logger := log.New(os.Stdout, "goapp ", log.LstdFlags|log.Lmsgprefix)

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		logger.Fatalf("failed to open database: %v", err)
	}
	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := telemetry.EnsureSchema(ctx, db, cfg.SchemaPath); err != nil {
		logger.Fatalf("failed to ensure schema: %v", err)
	}

	aiClient := ai.NewClient(cfg.HuggingFaceToken)

	app := fiber.New()
	handlers := api.NewHandlers(db, aiClient, logger)
	api.RegisterRoutes(app, handlers)

	go func() {
		if err := app.Listen(cfg.ListenAddress()); err != nil {
			logger.Fatalf("fiber server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.Shutdown(); err != nil {
		logger.Printf("error during Fiber shutdown: %v", err)
	}

	if err := db.Close(); err != nil {
		logger.Printf("error closing database: %v", err)
	}

	<-shutdownCtx.Done()
}
