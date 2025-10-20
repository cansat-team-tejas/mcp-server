package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"

	"goapp/internal/ai"
	"goapp/internal/questions"
	"goapp/internal/telemetry"
)

type Handlers struct {
	db            *sql.DB // Main database (legacy)
	ai            *ai.Client
	logger        *log.Logger
	currentDB     *sql.DB // Current working database
	currentDBPath string
	dbMutex       sync.RWMutex
}

func NewHandlers(db *sql.DB, aiClient *ai.Client, logger *log.Logger) Handlers {
	return Handlers{
		db:        db,
		ai:        aiClient,
		logger:    logger,
		currentDB: db, // Initially use main database
	}
}

func RegisterRoutes(app *fiber.App, h *Handlers) {
	app.Get("/chat", h.handleChat)
	app.Post("/ask", h.handleAsk)
	app.Get("/telemetry", h.handleTelemetry)
	app.Post("/query", h.handleQuery)
	app.Post("/telemetry/push", h.handlePushData)
	app.Post("/database/create", h.handleCreateDatabase)
	app.Get("/database/current", h.handleGetCurrentDatabase)
}

func (h *Handlers) handleChat(c *fiber.Ctx) error {
	question := c.Query("prompt")
	if strings.TrimSpace(question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "prompt query parameter is required"})
	}

	h.dbMutex.RLock()
	currentDB := h.currentDB
	h.dbMutex.RUnlock()

	// No DB/data guard removed: allow questions and command processing even without DB
	answer, err := questions.AnswerQuestion(requestContext(c), question, currentDB, h.ai)
	if err != nil {
		h.logger.Printf("chat endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(buildAnswerPayload(answer))
}

func (h *Handlers) handleAsk(c *fiber.Ctx) error {
	var req AskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}
	if strings.TrimSpace(req.Question) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "question is required"})
	}

	h.dbMutex.RLock()
	currentDB := h.currentDB
	h.dbMutex.RUnlock()

	// No DB/data guard: allow questions and command processing even without DB
	answer, err := questions.AnswerQuestion(requestContext(c), req.Question, currentDB, h.ai)
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

func (h *Handlers) handleTelemetry(c *fiber.Ctx) error {
	h.dbMutex.RLock()
	currentDB := h.currentDB
	h.dbMutex.RUnlock()

	if currentDB == nil {
		return c.JSON([]any{})
	}
	rows, err := telemetry.GetTelemetry(requestContext(c), currentDB)
	if err != nil {
		h.logger.Printf("telemetry endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handlers) handleQuery(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}

	trimmed := strings.TrimSpace(req.SQL)
	if !strings.HasPrefix(strings.ToLower(trimmed), "select") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Only SELECT queries allowed"})
	}

	h.dbMutex.RLock()
	currentDB := h.currentDB
	h.dbMutex.RUnlock()

	if currentDB == nil {
		return c.JSON(fiber.Map{"result": []any{}})
	}
	result, err := telemetry.Query(requestContext(c), currentDB, trimmed)
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

func (h *Handlers) handlePushData(c *fiber.Ctx) error {
	var row TelemetryRow
	if err := c.BodyParser(&row); err != nil {
		// Try uppercase schema
		var upper TelemetryRowUpper
		if err2 := c.BodyParser(&upper); err2 != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
		}
		row = upper.ToLowerCase()
	}

	h.dbMutex.RLock()
	currentDB := h.currentDB
	currentPath := h.currentDBPath
	h.dbMutex.RUnlock()

	if currentDB == nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "no active database; create one via /database/create first"})
	}

	query := `INSERT INTO telemetry (
		TEAM_ID, mission_time_s, packet_count, altitude, pressure, temperature, voltage,
		gnss_time, latitude, longitude, gps_altitude, satellites,
		accel_x, accel_y, accel_z, gyro_spin_rate, flight_state,
		gyro_x, gyro_y, gyro_z, roll, pitch, yaw,
		mag_x, mag_y, mag_z, humidity, current, power,
		baro_altitude, air_quality_raw, aq_ethanol_ppm, mcu_temp_c,
		rssi_dbm, health_flags, rtc_epoch, cmd_echo, log_data
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := currentDB.ExecContext(requestContext(c), query,
		row.TEAM_ID, row.MissionTimeS, row.PacketCount, row.Altitude, row.Pressure,
		row.Temperature, row.Voltage, row.GNSSTime, row.Latitude, row.Longitude,
		row.GPSAltitude, row.Satellites, row.AccelX, row.AccelY, row.AccelZ,
		row.GyroSpinRate, row.FlightState, row.GyroX, row.GyroY, row.GyroZ,
		row.Roll, row.Pitch, row.Yaw, row.MagX, row.MagY, row.MagZ,
		row.Humidity, row.Current, row.Power, row.BaroAltitude, row.AirQualityRaw,
		row.AQEthanolPPM, row.MCUTempC, row.RSSIDBm, row.HealthFlags,
		row.RTCEpoch, row.CMDEcho, row.LogData,
	)

	if err != nil {
		h.logger.Printf("push data error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id, err := result.LastInsertId()
	if err != nil {
		h.logger.Printf("get last insert id error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Data inserted successfully",
		"id":      id,
		"db_path": currentPath,
	})
}

func (h *Handlers) handleCreateDatabase(c *fiber.Ctx) error {
	var req CreateDatabaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}

	if strings.TrimSpace(req.DBPath) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "db_path is required"})
	}

	// Ensure the databases directory exists
	dbDir := "databases"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		h.logger.Printf("create databases directory error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Construct full path in databases directory
	dbPath := filepath.Join(dbDir, filepath.Base(req.DBPath))

	// Create new database connection
	newDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		h.logger.Printf("create database error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Configure the new database connection
	newDB.SetConnMaxLifetime(0)
	newDB.SetMaxOpenConns(1)
	newDB.SetMaxIdleConns(1)

	// Apply schema to the new database
	if err := telemetry.EnsureSchema(requestContext(c), newDB, ""); err != nil {
		newDB.Close()
		h.logger.Printf("apply schema error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Switch to the new database
	h.dbMutex.Lock()
	oldDB := h.currentDB
	h.currentDB = newDB
	h.currentDBPath = dbPath
	h.dbMutex.Unlock()

	// Close old database if it's not the main one
	if oldDB != h.db {
		oldDB.Close()
	}

	h.logger.Printf("Switched to new database: %s", dbPath)

	absPath, _ := filepath.Abs(dbPath)

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":    "Database created successfully and set as current",
		"db_path":    dbPath,
		"full_path":  absPath,
		"now_active": true,
	})
}

func (h *Handlers) handleGetCurrentDatabase(c *fiber.Ctx) error {
	h.dbMutex.RLock()
	currentPath := h.currentDBPath
	h.dbMutex.RUnlock()

	if currentPath == "" {
		currentPath = "none (no active database)"
	}

	return c.JSON(fiber.Map{
		"current_db_path": currentPath,
	})
}
