package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	"goapp/internal/ai"
	"goapp/internal/api"
	"goapp/internal/config"
	"goapp/internal/simulation"
	"goapp/internal/telemetry"

	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	logger := log.New(os.Stdout, "goapp ", log.LstdFlags|log.Lmsgprefix)

	var db *gorm.DB
	var err error
	if cfg.DBPath != "" {
		db, err = telemetry.OpenDatabase(cfg.DBPath)
		if err != nil {
			logger.Fatalf("failed to open database: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := telemetry.EnsureSchema(ctx, db, cfg.SchemaPath); err != nil {
			logger.Fatalf("failed to ensure schema: %v", err)
		}
		if cfg.ReadOnlyMode {
			if err := simulation.SeedDatabase(ctx, db); err != nil {
				logger.Fatalf("failed to build canonical simulation dataset: %v", err)
			}
		} else if err := telemetry.SeedSimulationData(ctx, db, cfg.SimulationSeedPath); err != nil {
			logger.Fatalf("failed to seed simulation data: %v", err)
		}
	} else {
		logger.Println("no default DB configured; starting without an active database")
	}

	aiClient := ai.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
		BodyLimit:             256 * 1024,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Cache-Control", "no-store")
		return c.Next()
	})

	// Simplified CORS: allow any origin to access the telemetry API.
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE, PUT")
		c.Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	// API key authentication — skip /health so Docker healthcheck still works
	if cfg.APIKey != "" {
		app.Use(func(c *fiber.Ctx) error {
			if c.Path() == "/health" {
				return c.Next()
			}
			if c.Get("X-API-Key") != cfg.APIKey {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			}
			return c.Next()
		})
	}

	app.Use(limiter.New(limiter.Config{
		Max:        120,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "global rate limit exceeded"})
		},
	}))

	// Strict limiter for AI requests.
	aiLimiter := limiter.New(limiter.Config{
		Max:        8,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "ai rate limit exceeded"})
		},
	})

	handlers := api.NewHandlers(db, cfg.DBPath, cfg.ReadOnlyMode, cfg.MaxTelemetryRows, aiClient, logger)
	api.RegisterRoutes(app, &handlers, aiLimiter, cfg.StaticDir)

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

	if err := handlers.Close(); err != nil {
		logger.Printf("error closing database: %v", err)
	}

	<-shutdownCtx.Done()
}
