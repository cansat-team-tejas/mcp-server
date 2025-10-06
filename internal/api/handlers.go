package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"goapp/internal/ai"
	"goapp/internal/questions"
	"goapp/internal/telemetry"
)

type Handlers struct {
	db     *sql.DB
	ai     *ai.Client
	logger *log.Logger
}

func NewHandlers(db *sql.DB, aiClient *ai.Client, logger *log.Logger) Handlers {
	return Handlers{db: db, ai: aiClient, logger: logger}
}

func RegisterRoutes(app *fiber.App, h Handlers) {
	app.Get("/chat", h.handleChat)
	app.Post("/ask", h.handleAsk)
	app.Get("/telemetry", h.handleTelemetry)
	app.Post("/query", h.handleQuery)
}

func (h Handlers) handleChat(c *fiber.Ctx) error {
	question := c.Query("prompt")
	if strings.TrimSpace(question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "prompt query parameter is required"})
	}

	answer, err := questions.AnswerQuestion(requestContext(c), question, h.db, h.ai)
	if err != nil {
		h.logger.Printf("chat endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(buildAnswerPayload(answer))
}

func (h Handlers) handleAsk(c *fiber.Ctx) error {
	var req AskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "question is required"})
	}

	answer, err := questions.AnswerQuestion(requestContext(c), req.Question, h.db, h.ai)
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

func (h Handlers) handleTelemetry(c *fiber.Ctx) error {
	rows, err := telemetry.GetTelemetry(requestContext(c), h.db)
	if err != nil {
		h.logger.Printf("telemetry endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h Handlers) handleQuery(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}

	trimmed := strings.TrimSpace(req.SQL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "select") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Only SELECT queries allowed"})
	}

	result, err := telemetry.Query(requestContext(c), h.db, trimmed)
	if err != nil {
		h.logger.Printf("query endpoint error: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"result": result})
}

func requestContext(c *fiber.Ctx) context.Context {
	if ctx := c.UserContext(); ctx != nil {
		return ctx
	}
	return context.Background()
}
