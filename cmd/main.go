package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"goapp/internal/ai"
	"goapp/internal/api"
	"goapp/internal/config"
	"goapp/internal/xbee"
)

func main() {
	cfg := config.Load()

	logger := log.New(os.Stdout, "goapp ", log.LstdFlags|log.Lmsgprefix)

	aiClient := ai.NewClient(cfg.LLMToken)

	// Initialize XBee service
	xbeeService, err := xbee.NewXBeeService(cfg.DBPath)
	if err != nil {
		logger.Fatalf("failed to initialize XBee service: %v", err)
	}
	logger.Println("XBee service initialized successfully")

	app := fiber.New(fiber.Config{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	})

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:")
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Upgrade, Connection, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Register existing API routes
	handlers := api.NewHandlers(aiClient, logger)
	api.RegisterRoutes(app, handlers)

	// Register AI chat handlers with streaming support
	aiChatHandlers := ai.NewChatHandlers(aiClient)
	aiChatHandlers.RegisterRoutes(app)

	// Register XBee API routes
	xbeeHandlers := xbee.NewXBeeHandlers(xbeeService)
	xbeeHandlers.RegisterRoutes(app)

	// Add WebSocket endpoint for XBee live streaming - simplified approach
	app.Get("/api/xbee/ws", func(c *fiber.Ctx) error {
		// For now, return an informational message
		// WebSocket integration will be handled separately
		return c.JSON(map[string]interface{}{
			"message": "WebSocket endpoint available",
			"note":    "Use WebSocket client to connect for live streaming",
			"stats":   xbeeService.GetStats(),
		})
	})

	go func() {
		if err := app.Listen(cfg.ListenAddress()); err != nil {
			logger.Fatalf("fiber server error: %v", err)
		}
	}()

	logger.Printf("Server started on %s", cfg.ListenAddress())
	logger.Println("AI Chat API: http://localhost:8000/api/chat")
	logger.Println("AI Chat WebSocket: ws://localhost:8000/api/chat/ws")
	logger.Println("XBee WebSocket endpoint: ws://localhost:8000/api/xbee/ws")
	logger.Println("XBee REST API: http://localhost:8000/api/xbee/")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.Shutdown(); err != nil {
		logger.Printf("error during Fiber shutdown: %v", err)
	}

	<-shutdownCtx.Done()
}
