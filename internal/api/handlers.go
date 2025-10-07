package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"goapp/internal/ai"
	"goapp/internal/models"
	"goapp/internal/questions"
	"goapp/internal/telemetry"
)

type Handlers struct {
	ai     *ai.Client
	logger *log.Logger
}

func NewHandlers(aiClient *ai.Client, logger *log.Logger) Handlers {
	return Handlers{ai: aiClient, logger: logger}
}

func RegisterRoutes(app *fiber.App, h Handlers) {
	app.Post("/create-db", h.handleCreateDB)
	app.Post("/ask", h.handleAsk)
	app.Post("/data", h.handleData)
	app.Post("/insert-data", h.handleInsertData)
}

func (h Handlers) handleAsk(c *fiber.Ctx) error {
	var req AskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "question is required"})
	}
	if strings.TrimSpace(req.Filename) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename is required"})
	}

	db, err := telemetry.EnsureSchema(req.Filename)
	var fallbackContext string
	if err != nil {
		h.logger.Printf("database not available, using context fallback: %v", err)
		// Try to load context from .txt file
		contextFile := strings.TrimSuffix(req.Filename, filepath.Ext(req.Filename)) + ".txt"
		if data, err := os.ReadFile(contextFile); err == nil {
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

func (h Handlers) handleCreateDB(c *fiber.Ctx) error {
	var req CreateDBRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Filename) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename is required"})
	}

	_, err := telemetry.EnsureSchema(req.Filename)
	if err != nil {
		h.logger.Printf("create db error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Database created successfully"})
}

func (h Handlers) handleData(c *fiber.Ctx) error {
	var req DataRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Filename) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename is required"})
	}

	db, err := telemetry.EnsureSchema(req.Filename)
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

func (h Handlers) handleInsertData(c *fiber.Ctx) error {
	var req InsertDataRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Filename) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename is required"})
	}

	db, err := telemetry.EnsureSchema(req.Filename)
	if err != nil {
		h.logger.Printf("open db error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	telemetryData := &models.Telemetry{
		TeamID:        req.TeamID,
		MissionTimeS:  req.MissionTimeS,
		PacketCount:   req.PacketCount,
		Altitude:      req.Altitude,
		Pressure:      req.Pressure,
		Temperature:   req.Temperature,
		Voltage:       req.Voltage,
		GnssTime:      req.GnssTime,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		GpsAltitude:   req.GpsAltitude,
		Satellites:    req.Satellites,
		AccelX:        req.AccelX,
		AccelY:        req.AccelY,
		AccelZ:        req.AccelZ,
		GyroSpinRate:  req.GyroSpinRate,
		FlightState:   req.FlightState,
		GyroX:         req.GyroX,
		GyroY:         req.GyroY,
		GyroZ:         req.GyroZ,
		Roll:          req.Roll,
		Pitch:         req.Pitch,
		Yaw:           req.Yaw,
		MagX:          req.MagX,
		MagY:          req.MagY,
		MagZ:          req.MagZ,
		Humidity:      req.Humidity,
		Current:       req.Current,
		Power:         req.Power,
		BaroAltitude:  req.BaroAltitude,
		AirQualityRaw: req.AirQualityRaw,
		AqEthanolPpm:  req.AqEthanolPpm,
		McuTempC:      req.McuTempC,
		RssiDbm:       req.RssiDbm,
		HealthFlags:   req.HealthFlags,
		RtcEpoch:      req.RtcEpoch,
		CmdEcho:       req.CmdEcho,
	}

	if err := db.Create(telemetryData).Error; err != nil {
		h.logger.Printf("insert data error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Data inserted successfully", "id": telemetryData.ID})
}
