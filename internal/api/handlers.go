package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"goapp/internal/ai"
	"goapp/internal/questions"
	"goapp/internal/telemetry"
)

type Handlers struct {
	ai         *ai.Client
	logger     *log.Logger
	missionDir string
}

func NewHandlers(aiClient *ai.Client, logger *log.Logger, missionDir string) Handlers {
	return Handlers{ai: aiClient, logger: logger, missionDir: missionDir}
}

func RegisterRoutes(app *fiber.App, h Handlers) {
	app.Post("/ask", h.handleAsk)
	app.Post("/data", h.handleData)
}

func (h Handlers) handleAsk(c *fiber.Ctx) error {
	var req AskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "question is required"})
	}
	dbPath, contextPath, err := h.resolveMissionPaths(req.Filename)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	db, err := telemetry.EnsureSchema(dbPath)
	var fallbackContext string
	if err != nil {
		h.logger.Printf("database not available, using context fallback: %v", err)
		// Try to load context from .txt file
		if data, err := os.ReadFile(contextPath); err == nil {
			fallbackContext = string(data)
		} else {
			fallbackContext = "No telemetry data available. This is a CanSat telemetry system for collecting and analyzing flight data."
		}
		db = nil // Set db to nil to indicate fallback mode
	}

	answer, err := questions.AnswerQuestion(c.Context(), req.Question, db, h.ai, fallbackContext)
	if err != nil {
		h.logger.Printf("ask endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(buildAnswerPayload(answer))
}

func buildAnswerPayload(answer questions.Answer) fiber.Map {
	payload := fiber.Map{
		"answer": fiber.Map{
			"content": answer.Content,
		},
	}

	if len(answer.Commands) == 1 {
		payload["command"] = answer.Commands[0]
	} else if len(answer.Commands) > 1 {
		payload["command"] = strings.Join(answer.Commands, ",")
	}

	return payload
}

func (h Handlers) handleData(c *fiber.Ctx) error {
	var req DataRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	dbPath, _, err := h.resolveMissionPaths(req.Filename)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	db, err := telemetry.EnsureSchema(dbPath)
	if err != nil {
		h.logger.Printf("open db error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	rows, err := telemetry.GetTelemetry(db)
	if err != nil {
		h.logger.Printf("data endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h Handlers) resolveMissionPaths(filename string) (string, string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", "", errors.New("filename is required")
	}

	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) {
		return "", "", errors.New("invalid filename")
	}

	if !strings.HasSuffix(strings.ToLower(base), ".db") {
		base += ".db"
	}

	missionDir := h.missionDir
	if missionDir == "" {
		missionDir = "missions"
	}

	missionDir = filepath.Clean(missionDir)
	if err := os.MkdirAll(missionDir, 0o755); err != nil {
		return "", "", fmt.Errorf("prepare mission directory: %w", err)
	}

	dbPath := filepath.Join(missionDir, base)
	contextPath := strings.TrimSuffix(dbPath, filepath.Ext(dbPath)) + ".txt"
	return dbPath, contextPath, nil
}
