package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2/middleware/filesystem"

	"github.com/gofiber/fiber/v2"

	"goapp/internal/ai"
	"goapp/internal/questions"
	"goapp/internal/telemetry"

	"gorm.io/gorm"
)

type Handlers struct {
	db               *gorm.DB
	defaultDBPath    string
	readOnlyMode     bool
	maxTelemetryRows int
	ai               *ai.Client
	logger           *log.Logger
	currentDB        *gorm.DB
	currentDBPath    string
	dbMutex          sync.RWMutex
}

func NewHandlers(db *gorm.DB, defaultDBPath string, readOnlyMode bool, maxTelemetryRows int, aiClient *ai.Client, logger *log.Logger) Handlers {
	return Handlers{
		db:               db,
		defaultDBPath:    defaultDBPath,
		readOnlyMode:     readOnlyMode,
		maxTelemetryRows: maxTelemetryRows,
		ai:               aiClient,
		logger:           logger,
		currentDB:        db,
		currentDBPath:    defaultDBPath,
	}
}

func (h *Handlers) Close() error {
	h.dbMutex.Lock()
	defer h.dbMutex.Unlock()

	if h.currentDB != nil && h.currentDB != h.db {
		if err := telemetry.CloseDatabase(h.currentDB); err != nil {
			return err
		}
	}
	if h.db != nil {
		if err := telemetry.CloseDatabase(h.db); err != nil {
			return err
		}
	}
	return nil
}

// safeDBNameRe only allows simple filename-safe database names like "mission_001.db"
var safeDBNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*\.db$`)

// validateSelectQuery rejects dangerous SQL patterns before passing to SQLite.
func validateSelectQuery(sql string) error {
	if len(sql) > 2000 {
		return fmt.Errorf("query exceeds maximum allowed length")
	}
	lower := strings.ToLower(sql)
	banned := []string{";", "--", "/*", "*/", "attach ", "detach ", "pragma "}
	for _, b := range banned {
		if strings.Contains(lower, b) {
			return fmt.Errorf("query contains disallowed syntax")
		}
	}
	return nil
}

func RegisterRoutes(app *fiber.App, h *Handlers, aiLimiter fiber.Handler, staticDir string) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post("/ask", aiLimiter, h.handleAsk)

	app.Get("/telemetry", h.handleTelemetry)
	app.Post("/query", h.handleQuery)
	app.Post("/telemetry/push", h.handlePushData)
	app.Post("/database/create", h.handleCreateDatabase)
	app.Get("/database/current", h.handleGetCurrentDatabase)

	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			app.Use("/", filesystem.New(filesystem.Config{
				Root:         http.Dir(staticDir),
				Browse:       false,
				Index:        "index.html",
				NotFoundFile: "index.html",
			}))
		}
	}
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

	answer, err := questions.AnswerQuestion(requestContext(c), req.Question, currentDB, h.ai, req.CurrentRow)
	if err != nil {
		h.logger.Printf("ask endpoint error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to process question"})
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
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to retrieve telemetry"})
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
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "only SELECT queries are allowed"})
	}
	if err := validateSelectQuery(trimmed); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query execution failed"})
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
	if h.readOnlyMode {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "telemetry writes are disabled in shared simulation mode"})
	}

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

	record := telemetry.Record{
		TeamID:        row.TEAM_ID,
		MissionTimeS:  row.MissionTimeS,
		PacketCount:   row.PacketCount,
		Altitude:      row.Altitude,
		Pressure:      row.Pressure,
		Temperature:   row.Temperature,
		Voltage:       row.Voltage,
		GNSSTime:      row.GNSSTime,
		Latitude:      row.Latitude,
		Longitude:     row.Longitude,
		GPSAltitude:   row.GPSAltitude,
		Satellites:    row.Satellites,
		AccelX:        row.AccelX,
		AccelY:        row.AccelY,
		AccelZ:        row.AccelZ,
		GyroSpinRate:  row.GyroSpinRate,
		FlightState:   row.FlightState,
		GyroX:         row.GyroX,
		GyroY:         row.GyroY,
		GyroZ:         row.GyroZ,
		Roll:          row.Roll,
		Pitch:         row.Pitch,
		Yaw:           row.Yaw,
		MagX:          row.MagX,
		MagY:          row.MagY,
		MagZ:          row.MagZ,
		Humidity:      row.Humidity,
		Current:       row.Current,
		Power:         row.Power,
		BaroAltitude:  row.BaroAltitude,
		AirQualityRaw: row.AirQualityRaw,
		AQEthanolPPM:  row.AQEthanolPPM,
		MCUTempC:      row.MCUTempC,
		RSSIDBm:       row.RSSIDBm,
		HealthFlags:   row.HealthFlags,
		RTCEpoch:      row.RTCEpoch,
		CMDEcho:       row.CMDEcho,
		LogData:       row.LogData,
	}

	if err := currentDB.WithContext(requestContext(c)).Create(&record).Error; err != nil {
		h.logger.Printf("push data error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store telemetry data"})
	}
	if err := telemetry.TrimTelemetryHistory(requestContext(c), currentDB, h.maxTelemetryRows); err != nil {
		h.logger.Printf("trim telemetry history error: %v", err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Data inserted successfully",
		"id":      record.ID,
		"db_path": currentPath,
	})
}

func (h *Handlers) handleCreateDatabase(c *fiber.Ctx) error {
	if h.readOnlyMode {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{"error": "database creation is disabled in shared simulation mode"})
	}

	var req CreateDatabaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON payload"})
	}

	if strings.TrimSpace(req.DBPath) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "db_path is required"})
	}

	// Validate that the db name is safe (alphanumeric/hyphen/underscore + .db)
	baseName := filepath.Base(req.DBPath)
	if !safeDBNameRe.MatchString(baseName) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "db_path must be a simple .db filename (letters, digits, hyphens, underscores)"})
	}

	dbDir := "databases"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		h.logger.Printf("create databases directory error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create database directory"})
	}

	dbPath := filepath.Join(dbDir, baseName)

	newDB, err := telemetry.OpenDatabase(dbPath)
	if err != nil {
		h.logger.Printf("create database error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open database"})
	}

	// Apply schema to the new database
	if err := telemetry.EnsureSchema(requestContext(c), newDB, ""); err != nil {
		_ = telemetry.CloseDatabase(newDB)
		h.logger.Printf("apply schema error: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to initialize database schema"})
	}

	// Switch to the new database
	h.dbMutex.Lock()
	oldDB := h.currentDB
	h.currentDB = newDB
	h.currentDBPath = dbPath
	h.dbMutex.Unlock()

	if oldDB != h.db {
		_ = telemetry.CloseDatabase(oldDB)
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
		"read_only":       h.readOnlyMode,
	})
}
