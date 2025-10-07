// Package xbee provides HTTP handlers for XBee radio communication API endpoints.
// These handlers expose REST API and WebSocket interfaces for CanSat mission control,
// including connection management, command transmission, data retrieval, and conversation history.
package xbee

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// XBeeHandlers provides HTTP handlers for all XBee-related API endpoints.
// It wraps the XBeeService to provide RESTful API access to XBee functionality.
type XBeeHandlers struct {
	service *XBeeService
}

func NewXBeeHandlers(service *XBeeService) *XBeeHandlers {
	return &XBeeHandlers{
		service: service,
	}
}

// WebSocket endpoint for live streaming
func (h *XBeeHandlers) HandleWebSocket(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusUpgradeRequired, "WebSocket connection required")
}

// REST API Handlers

// GET /api/xbee/ports - List available serial ports
func (h *XBeeHandlers) ListPorts(c *fiber.Ctx) error {
	response := h.service.ListPorts()
	return c.JSON(response)
}

// POST /api/xbee/connect - Connect to XBee
func (h *XBeeHandlers) Connect(c *fiber.Ctx) error {
	var req struct {
		Port   string       `json:"port"`
		Config SerialConfig `json:"config"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	response := h.service.OpenPort(req.Port, req.Config)
	if !response.Success {
		return c.Status(500).JSON(response)
	}

	return c.JSON(response)
}

// POST /api/xbee/disconnect - Disconnect from XBee
func (h *XBeeHandlers) Disconnect(c *fiber.Ctx) error {
	response := h.service.ClosePort()
	if !response.Success {
		return c.Status(500).JSON(response)
	}

	return c.JSON(response)
}

// GET /api/xbee/status - Get connection status
func (h *XBeeHandlers) GetStatus(c *fiber.Ctx) error {
	status := h.service.serialManager.GetStatus()
	mission := h.service.GetCurrentMission()
	health := h.service.GetConnectionHealth()

	return c.JSON(map[string]interface{}{
		"success":           true,
		"connection":        status,
		"connection_health": health,
		"mission":           mission,
		"stats":             h.service.GetStats(),
	})
}

// POST /api/xbee/command - Send command to XBee
func (h *XBeeHandlers) SendCommand(c *fiber.Ctx) error {
	var req struct {
		Command string `json:"command"`
		Data    string `json:"data,omitempty"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Command == "" {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Command is required",
		})
	}

	var data []byte
	if req.Data != "" {
		data = []byte(req.Data)
	}

	err := h.service.SendCommand(req.Command, data)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(Response{Success: true})
}

// POST /api/xbee/mission/start - Start new mission
func (h *XBeeHandlers) StartMission(c *fiber.Ctx) error {
	var req struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("Mission_%s",
			h.service.stats.StartTime.Format("2006-01-02_15-04-05"))
	}

	err := h.service.StartNewMission(req.Name)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(map[string]interface{}{
		"success": true,
		"mission": h.service.GetCurrentMission(),
	})
}

// GET /api/xbee/mission - Get current mission info
func (h *XBeeHandlers) GetMission(c *fiber.Ctx) error {
	mission := h.service.GetCurrentMission()
	return c.JSON(map[string]interface{}{
		"success": true,
		"mission": mission,
	})
}

// GET /api/xbee/stats - Get current statistics
func (h *XBeeHandlers) GetStats(c *fiber.Ctx) error {
	stats := h.service.GetStats()
	return c.JSON(map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// GET /api/xbee/telemetry - Get telemetry data
func (h *XBeeHandlers) GetTelemetry(c *fiber.Ctx) error {
	// Parse query parameters
	limitStr := c.Query("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	var telemetries []interface{}
	var dbErr error

	if startTimeStr != "" && endTimeStr != "" {
		// Get by time range
		startTime, err1 := strconv.ParseInt(startTimeStr, 10, 64)
		endTime, err2 := strconv.ParseInt(endTimeStr, 10, 64)

		if err1 != nil || err2 != nil {
			return c.Status(400).JSON(Response{
				Success: false,
				Error:   "Invalid time format. Use Unix timestamp.",
			})
		}

		data, err := h.service.GetTelemetryByTimeRange(startTime, endTime)
		if err == nil {
			for _, item := range data {
				telemetries = append(telemetries, item)
			}
		}
		dbErr = err
	} else {
		// Get latest records
		data, err := h.service.GetLatestTelemetry(limit)
		if err == nil {
			for _, item := range data {
				telemetries = append(telemetries, item)
			}
		}
		dbErr = err
	}

	if dbErr != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   fmt.Sprintf("Database error: %v", dbErr),
		})
	}

	return c.JSON(map[string]interface{}{
		"success": true,
		"data":    telemetries,
		"count":   len(telemetries),
	})
}

// GET /api/xbee/activity - Get activity log
func (h *XBeeHandlers) GetActivity(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	activities := h.service.GetActivityLog(limit)

	return c.JSON(map[string]interface{}{
		"success":    true,
		"activities": activities,
		"count":      len(activities),
	})
}

// GET /api/xbee/telemetry/stats - Get telemetry statistics
func (h *XBeeHandlers) GetTelemetryStats(c *fiber.Ctx) error {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "start_time and end_time parameters are required",
		})
	}

	startTime, err1 := strconv.ParseInt(startTimeStr, 10, 64)
	endTime, err2 := strconv.ParseInt(endTimeStr, 10, 64)

	if err1 != nil || err2 != nil {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Invalid time format. Use Unix timestamp.",
		})
	}

	stats, err := h.service.GetTelemetryStatsByTimeRange(startTime, endTime)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   fmt.Sprintf("Database error: %v", err),
		})
	}

	return c.JSON(map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// WebSocket upgrade handler for Fiber
func (h *XBeeHandlers) UpgradeWebSocket(c *fiber.Ctx) error {
	// Check for WebSocket upgrade headers
	if c.Get("Upgrade") == "websocket" && c.Get("Connection") == "Upgrade" {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// GET /api/xbee/conversation/:missionId - Get conversation history for a mission
func (h *XBeeHandlers) GetConversationHistory(c *fiber.Ctx) error {
	missionID := c.Params("missionId")
	if missionID == "" {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Mission ID is required",
		})
	}

	// Parse optional query parameters
	limitStr := c.Query("limit", "100")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get conversation history
	conversations, err := h.service.GetConversationHistoryPaginated(missionID, limit, offset)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to get conversation history: %v", err),
		})
	}

	// Get total count for pagination
	totalCount, err := h.service.GetConversationHistoryCount(missionID)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to get conversation count: %v", err),
		})
	}

	return c.JSON(map[string]interface{}{
		"success":       true,
		"conversations": conversations,
		"total_count":   totalCount,
		"limit":         limit,
		"offset":        offset,
		"has_more":      int64(offset+len(conversations)) < totalCount,
	})
}

// GET /api/xbee/conversation/:missionId/all - Get all conversation history for a mission
func (h *XBeeHandlers) GetAllConversationHistory(c *fiber.Ctx) error {
	missionID := c.Params("missionId")
	if missionID == "" {
		return c.Status(400).JSON(Response{
			Success: false,
			Error:   "Mission ID is required",
		})
	}

	conversations, err := h.service.GetConversationHistory(missionID)
	if err != nil {
		return c.Status(500).JSON(Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to get conversation history: %v", err),
		})
	}

	return c.JSON(map[string]interface{}{
		"success":       true,
		"conversations": conversations,
		"total_count":   len(conversations),
	})
}

// GET /api/xbee/health - Get detailed connection health information
func (h *XBeeHandlers) GetConnectionHealth(c *fiber.Ctx) error {
	health := h.service.GetConnectionHealth()

	return c.JSON(map[string]interface{}{
		"success": true,
		"health":  health,
	})
}

// Register all routes
func (h *XBeeHandlers) RegisterRoutes(app *fiber.App) {
	xbee := app.Group("/api/xbee")

	// Connection management
	xbee.Get("/ports", h.ListPorts)
	xbee.Post("/connect", h.Connect)
	xbee.Post("/disconnect", h.Disconnect)
	xbee.Get("/status", h.GetStatus)
	xbee.Get("/health", h.GetConnectionHealth)

	// Command sending
	xbee.Post("/command", h.SendCommand)

	// Mission management
	xbee.Post("/mission/start", h.StartMission)
	xbee.Get("/mission", h.GetMission)

	// Data retrieval
	xbee.Get("/stats", h.GetStats)
	xbee.Get("/telemetry", h.GetTelemetry)
	xbee.Get("/telemetry/stats", h.GetTelemetryStats)
	xbee.Get("/activity", h.GetActivity)

	// Conversation history
	xbee.Get("/conversation/:missionId", h.GetConversationHistory)
	xbee.Get("/conversation/:missionId/all", h.GetAllConversationHistory)

	// WebSocket endpoint (will be handled separately)
	// xbee.Get("/ws", websocket.New(h.HandleWebSocketConnection))
}

// Helper function to create WebSocket response
func createWebSocketResponse(msgType string, success bool, data interface{}, err string) map[string]interface{} {
	response := map[string]interface{}{
		"type":    msgType,
		"success": success,
	}

	if data != nil {
		response["data"] = data
	}

	if err != "" {
		response["error"] = err
	}

	return response
}
