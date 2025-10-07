// Package xbee provides comprehensive XBee radio communication capabilities for CanSat missions.
// It handles serial communication, frame processing, real-time telemetry streaming,
// mission management, and conversation history tracking.
//
// Key features:
// - Bidirectional XBee communication with frame-based protocol
// - Automatic mission creation and per-mission database isolation
// - Real-time WebSocket streaming for live telemetry data
// - Complete conversation history tracking (commands, telemetry, responses, errors)
// - Statistics monitoring and activity logging
// - Thread-safe operations with proper mutex protection
package xbee

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"goapp/internal/models"
	"goapp/internal/telemetry"
)

// XBeeService is the main service for managing XBee radio communication with CanSat.
// It provides thread-safe operations for serial communication, mission management,
// real-time data streaming, and conversation history tracking.
type XBeeService struct {
	serialManager  *SerialManager
	frameProcessor *FrameProcessor

	// In-memory storage for now (to avoid CGO issues)
	telemetryData []models.Telemetry
	dataMutex     sync.RWMutex

	// WebSocket connections for live streaming
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	upgrader     websocket.Upgrader

	// Statistics tracking
	stats      *ServiceStats
	statsMutex sync.RWMutex

	// Mission management
	currentMission *MissionInfo
	missionMutex   sync.RWMutex

	// Activity log
	activityLog   []ActivityItem
	activityMutex sync.RWMutex

	// Database integration
	database *gorm.DB
	dbPath   string
}

type ServiceStats struct {
	PacketRate       float64    `json:"packetRate"` // Hz
	PacketsReceived  int        `json:"packetsReceived"`
	PacketsSent      int        `json:"packetsSent"`
	LastUpdate       time.Time  `json:"lastUpdate"`
	StartTime        time.Time  `json:"startTime"`
	LastCommand      string     `json:"lastCommand"`
	LastCommandEcho  string     `json:"lastCommandEcho"`
	FrameStats       FrameStats `json:"frameStats"`
	ConnectionStatus string     `json:"connectionStatus"` // "connected", "disconnected", "error"
	LastDataReceived time.Time  `json:"lastDataReceived"` // When we last received any data
	ConnectionUptime time.Time  `json:"connectionUptime"` // When connection was established
}

type MissionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"startTime"`
	IsActive  bool      `json:"isActive"`
	DBPath    string    `json:"dbPath"`
}

type LiveTelemetryData struct {
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      *models.Telemetry `json:"data,omitempty"`
	Stats     *ServiceStats     `json:"stats,omitempty"`
	Activity  *ActivityItem     `json:"activity,omitempty"`
	Error     string            `json:"error,omitempty"`
}

func NewXBeeService(dbPath string) (*XBeeService, error) {
	// Initialize database using existing telemetry package
	db, err := telemetry.EnsureSchema(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	service := &XBeeService{
		serialManager:  NewSerialManager(),
		frameProcessor: NewFrameProcessor(),
		database:       db,
		dbPath:         dbPath,
		clients:        make(map[*websocket.Conn]bool),
		activityLog:    make([]ActivityItem, 0, 1000),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure properly for production
			},
		},
		stats: &ServiceStats{
			StartTime:        time.Now(),
			ConnectionStatus: "disconnected", // Default status
		},
	}

	// Set up frame processor handlers
	service.frameProcessor.SetFrameHandler(service.handleIncomingFrame)
	service.frameProcessor.SetErrorHandler(service.handleFrameError)

	// Set up serial manager handlers
	service.serialManager.SetDataHandler(service.frameProcessor.ProcessByte)
	service.serialManager.SetDisconnectHandler(service.handleDisconnection)

	// Start statistics update goroutine
	go service.updateStatsLoop()

	return service, nil
}

// Mission Management
func (xs *XBeeService) StartNewMission(missionName string) error {
	xs.missionMutex.Lock()
	defer xs.missionMutex.Unlock()

	// Stop current mission if active
	if xs.currentMission != nil && xs.currentMission.IsActive {
		xs.currentMission.IsActive = false
	}

	// Create new mission
	missionID := fmt.Sprintf("mission_%d", time.Now().Unix())
	missionDBPath := fmt.Sprintf("missions/%s.db", missionID)

	// Create new database for mission using existing telemetry package
	missionDB, err := telemetry.EnsureSchema(missionDBPath)
	if err != nil {
		return fmt.Errorf("failed to create mission database: %v", err)
	}

	// Update service to use mission database
	xs.database = missionDB
	xs.dbPath = missionDBPath

	xs.currentMission = &MissionInfo{
		ID:        missionID,
		Name:      missionName,
		StartTime: time.Now(),
		IsActive:  true,
		DBPath:    missionDBPath,
	}

	// Reset statistics
	xs.statsMutex.Lock()
	xs.stats = &ServiceStats{
		StartTime: time.Now(),
	}
	xs.statsMutex.Unlock()

	xs.addActivity("MISSION", "NEW_MISSION", fmt.Sprintf("Started new mission: %s", missionName))

	return nil
}

func (xs *XBeeService) GetCurrentMission() *MissionInfo {
	xs.missionMutex.RLock()
	defer xs.missionMutex.RUnlock()

	if xs.currentMission == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	mission := *xs.currentMission
	return &mission
}

// Serial Connection Management
func (xs *XBeeService) ListPorts() Response {
	ports, err := xs.serialManager.ListPorts()
	if err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to list ports: %v", err),
		}
	}

	return Response{
		Success: true,
		Ports:   ports,
	}
}

func (xs *XBeeService) OpenPort(portPath string, config SerialConfig) Response {
	err := xs.serialManager.Open(portPath, config)
	if err != nil {
		// Update connection status
		xs.statsMutex.Lock()
		xs.stats.ConnectionStatus = "error"
		xs.stats.LastUpdate = time.Now()
		xs.statsMutex.Unlock()

		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to open port: %v", err),
		}
	}

	// Update connection status
	xs.statsMutex.Lock()
	xs.stats.ConnectionStatus = "connected"
	xs.stats.ConnectionUptime = time.Now()
	xs.stats.LastUpdate = time.Now()
	xs.statsMutex.Unlock()

	xs.addActivity("CONNECTION", "CONNECTED", fmt.Sprintf("Connected to %s", portPath))
	xs.broadcastConnectionStatus(true)

	return Response{Success: true}
}

func (xs *XBeeService) ClosePort() Response {
	err := xs.serialManager.Close()
	if err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to close port: %v", err),
		}
	}

	// Update connection status
	xs.statsMutex.Lock()
	xs.stats.ConnectionStatus = "disconnected"
	xs.stats.LastUpdate = time.Now()
	xs.statsMutex.Unlock()

	xs.addActivity("CONNECTION", "DISCONNECTED", "Disconnected from XBee")
	xs.broadcastConnectionStatus(false)

	return Response{Success: true}
}

// Conversation History Management
func (xs *XBeeService) storeConversation(messageType, direction, content, source, metadata string) {
	// Only store conversations if we have an active mission
	xs.missionMutex.RLock()
	currentMission := xs.currentMission
	xs.missionMutex.RUnlock()

	if currentMission == nil || !currentMission.IsActive {
		return // No active mission to store conversation for
	}

	conversation := &models.ConversationHistory{
		MissionID:   currentMission.ID,
		Timestamp:   time.Now(),
		MessageType: messageType,
		Direction:   direction,
		Content:     content,
		Source:      source,
		Metadata:    metadata,
	}

	// Store in database
	if err := xs.database.Create(conversation).Error; err != nil {
		log.Printf("Failed to store conversation history: %v", err)
	}
}

// Command Handling
func (xs *XBeeService) SendCommand(command string, data []byte) error {
	if !xs.serialManager.IsOpen() {
		return fmt.Errorf("serial port not connected")
	}

	// Handle special START command for new missions
	if command == "START" {
		missionName := fmt.Sprintf("Mission_%s", time.Now().Format("2006-01-02_15-04-05"))
		if err := xs.StartNewMission(missionName); err != nil {
			return fmt.Errorf("failed to start new mission: %v", err)
		}
	}

	// Send AT command or data packet based on command type
	var err error
	if len(command) == 2 { // AT Command
		err = xs.frameProcessor.sendATCommand(command, data)
		xs.updateStats("command_sent", command)
	} else { // Data packet
		// For data packets, we might need to specify destination addresses
		// For now, use broadcast addresses
		err = xs.frameProcessor.sendDataPacket(append([]byte(command), data...), 0x0000000000000000, 0xFFFE)
		xs.updateStats("data_sent", command)
	}

	if err != nil {
		return err
	}

	xs.statsMutex.Lock()
	xs.stats.LastCommand = command
	xs.stats.PacketsSent++
	xs.statsMutex.Unlock()

	xs.addActivity("FRAME_SENT", command, fmt.Sprintf("Sent command: %s", command))

	// Store conversation history for commands
	xs.storeConversation("command", "sent", command, "gui", fmt.Sprintf(`{"data": "%x"}`, data))

	return nil
}

// Frame Handling
func (xs *XBeeService) handleIncomingFrame(frame XBeeFrameData) error {
	xs.statsMutex.Lock()
	xs.stats.PacketsReceived++
	xs.stats.LastUpdate = time.Now()
	xs.stats.LastDataReceived = time.Now() // Track when we last received data

	// Update frame-specific statistics
	switch frame.PacketType {
	case "TELEMETRY":
		xs.stats.FrameStats.TelemetryCount++
	case "LOG":
		xs.stats.FrameStats.LogEntryCount++
	case "CMD_RESPONSE":
		xs.stats.FrameStats.CommandEchoCount++
		xs.stats.LastCommandEcho = frame.Data
	default:
		xs.stats.FrameStats.UnknownCount++
	}
	xs.statsMutex.Unlock()

	// Parse and store telemetry data
	if frame.PacketType == "TELEMETRY" {
		telemetry, err := xs.parseTelemetryData(frame.Data)
		if err != nil {
			log.Printf("Failed to parse telemetry: %v", err)
			xs.addActivity("ERROR", "PARSE_ERROR", fmt.Sprintf("Failed to parse telemetry: %v", err))
		} else {
			// Store in database using GORM directly
			if err := xs.database.Create(telemetry).Error; err != nil {
				log.Printf("Failed to store telemetry: %v", err)
				xs.addActivity("ERROR", "DB_ERROR", fmt.Sprintf("Failed to store telemetry: %v", err))
			} else {
				// Broadcast live telemetry
				xs.broadcastLiveTelemetry(telemetry)

				// Store conversation history for telemetry
				xs.storeConversation("telemetry", "received", frame.Data, "xbee", fmt.Sprintf(`{"packet_type": "%s"}`, frame.PacketType))
			}
		}
	}

	xs.addActivity("FRAME_RECEIVED", frame.Type, fmt.Sprintf("Received %s frame", frame.Type))

	// Store conversation history for non-telemetry frames
	if frame.PacketType != "TELEMETRY" {
		messageType := "response"
		if frame.PacketType == "LOG" {
			messageType = "log"
		}
		xs.storeConversation(messageType, "received", frame.Data, "xbee", fmt.Sprintf(`{"packet_type": "%s", "frame_type": "%s"}`, frame.PacketType, frame.Type))
	}

	return nil
}

func (xs *XBeeService) handleFrameError(err error) {
	log.Printf("XBee frame error: %v", err)
	xs.addActivity("ERROR", "FRAME_ERROR", err.Error())

	// Store conversation history for errors
	xs.storeConversation("error", "received", err.Error(), "system", `{"error_type": "frame_error"}`)
}

// Handle XBee disconnection events
func (xs *XBeeService) handleDisconnection() {
	log.Printf("XBee disconnected unexpectedly")

	// Update connection status
	xs.statsMutex.Lock()
	xs.stats.ConnectionStatus = "disconnected"
	xs.stats.LastUpdate = time.Now()
	xs.statsMutex.Unlock()

	// Add activity log entry
	xs.addActivity("CONNECTION", "DISCONNECTED", "XBee connection lost unexpectedly")

	// Store conversation history for disconnection
	xs.storeConversation("error", "received", "XBee connection lost", "system", `{"error_type": "connection_lost", "automatic": true}`)

	// Broadcast disconnection status to connected clients
	xs.broadcastConnectionStatus(false)
}

// Telemetry Parsing (CSV format expected)
func (xs *XBeeService) parseTelemetryData(data string) (*models.Telemetry, error) {
	// Expected CSV format: TEAM_ID,mission_time_s,packet_count,altitude,pressure,temperature,voltage,gnss_time,latitude,longitude,gps_altitude,satellites,accel_x,accel_y,accel_z,gyro_spin_rate,flight_state,gyro_x,gyro_y,gyro_z,roll,pitch,yaw,mag_x,mag_y,mag_z,humidity,current,power,baro_altitude,air_quality_raw,aq_ethanol_ppm,mcu_temp_c,rssi_dbm,health_flags,rtc_epoch,cmd_echo

	// Simple CSV parsing (you might want to use a proper CSV library)
	fields := parseCSV(data)
	if len(fields) < 10 { // Minimum required fields
		return nil, fmt.Errorf("insufficient telemetry fields: got %d", len(fields))
	}

	telemetry := &models.Telemetry{}

	// Parse each field with error handling
	if len(fields) > 0 && fields[0] != "" {
		teamID := fields[0]
		telemetry.TeamID = &teamID
	}

	if len(fields) > 1 && fields[1] != "" {
		if val, err := strconv.ParseFloat(fields[1], 64); err == nil {
			telemetry.MissionTimeS = &val
		}
	}

	if len(fields) > 2 && fields[2] != "" {
		if val, err := strconv.Atoi(fields[2]); err == nil {
			telemetry.PacketCount = &val
		}
	}

	if len(fields) > 3 && fields[3] != "" {
		if val, err := strconv.ParseFloat(fields[3], 64); err == nil {
			telemetry.Altitude = &val
		}
	}

	if len(fields) > 4 && fields[4] != "" {
		if val, err := strconv.ParseFloat(fields[4], 64); err == nil {
			telemetry.Pressure = &val
		}
	}

	if len(fields) > 5 && fields[5] != "" {
		if val, err := strconv.ParseFloat(fields[5], 64); err == nil {
			telemetry.Temperature = &val
		}
	}

	// Add timestamp
	rtcEpoch := int(time.Now().Unix())
	telemetry.RtcEpoch = &rtcEpoch

	return telemetry, nil
}

// Statistics Management
func (xs *XBeeService) updateStatsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastPacketCount := 0

	for range ticker.C {
		xs.statsMutex.Lock()
		currentPackets := xs.stats.PacketsReceived

		// Calculate packet rate (packets per second)
		if currentPackets > lastPacketCount {
			xs.stats.PacketRate = float64(currentPackets - lastPacketCount)
		} else {
			xs.stats.PacketRate = 0.0
		}

		lastPacketCount = currentPackets
		xs.statsMutex.Unlock()

		// Broadcast stats update
		xs.broadcastStats()
	}
}

func (xs *XBeeService) updateStats(statType, command string) {
	xs.statsMutex.Lock()
	defer xs.statsMutex.Unlock()

	xs.stats.LastUpdate = time.Now()
	if statType == "command_sent" || statType == "data_sent" {
		xs.stats.LastCommand = command
	}
}

func (xs *XBeeService) GetStats() *ServiceStats {
	xs.statsMutex.RLock()
	defer xs.statsMutex.RUnlock()

	// Return a copy
	stats := *xs.stats
	return &stats
}

// Activity Management
func (xs *XBeeService) addActivity(actType, frameType, details string) {
	xs.activityMutex.Lock()
	defer xs.activityMutex.Unlock()

	activity := ActivityItem{
		Timestamp: time.Now(),
		Type:      actType,
		FrameType: frameType,
		Details:   details,
	}

	xs.activityLog = append(xs.activityLog, activity)

	// Keep only last 1000 activities
	if len(xs.activityLog) > 1000 {
		xs.activityLog = xs.activityLog[1:]
	}

	// Broadcast activity
	xs.broadcastActivity(activity)
}

func (xs *XBeeService) GetActivityLog(limit int) []ActivityItem {
	xs.activityMutex.RLock()
	defer xs.activityMutex.RUnlock()

	if limit <= 0 || limit > len(xs.activityLog) {
		limit = len(xs.activityLog)
	}

	// Return last 'limit' activities
	start := len(xs.activityLog) - limit
	if start < 0 {
		start = 0
	}

	activities := make([]ActivityItem, limit)
	copy(activities, xs.activityLog[start:])

	return activities
}

// WebSocket Broadcasting
func (xs *XBeeService) broadcastLiveTelemetry(telemetry *models.Telemetry) {
	data := LiveTelemetryData{
		Type:      "live_telemetry",
		Timestamp: time.Now(),
		Data:      telemetry,
	}
	xs.broadcastToClients(data)
}

// Connection Health Check
func (xs *XBeeService) GetConnectionHealth() map[string]interface{} {
	xs.statsMutex.RLock()
	defer xs.statsMutex.RUnlock()

	isConnected := xs.serialManager.IsOpen()
	timeSinceLastData := time.Since(xs.stats.LastDataReceived)

	health := map[string]interface{}{
		"is_connected":            isConnected,
		"connection_status":       xs.stats.ConnectionStatus,
		"time_since_last_data_ms": timeSinceLastData.Milliseconds(),
		"last_data_received":      xs.stats.LastDataReceived,
		"connection_uptime":       xs.stats.ConnectionUptime,
		"packets_received":        xs.stats.PacketsReceived,
		"packets_sent":            xs.stats.PacketsSent,
	}

	// Determine if connection seems healthy
	if isConnected {
		if timeSinceLastData > 30*time.Second {
			health["health_status"] = "poor"
			health["health_reason"] = "No data received for over 30 seconds"
		} else if timeSinceLastData > 10*time.Second {
			health["health_status"] = "warning"
			health["health_reason"] = "No data received for over 10 seconds"
		} else {
			health["health_status"] = "good"
			health["health_reason"] = "Receiving data regularly"
		}
	} else {
		health["health_status"] = "disconnected"
		health["health_reason"] = "XBee is not connected"
	}

	return health
}

func (xs *XBeeService) broadcastStats() {
	stats := xs.GetStats()
	data := LiveTelemetryData{
		Type:      "stats_update",
		Timestamp: time.Now(),
		Stats:     stats,
	}
	xs.broadcastToClients(data)
}

func (xs *XBeeService) broadcastActivity(activity ActivityItem) {
	data := LiveTelemetryData{
		Type:      "activity",
		Timestamp: time.Now(),
		Activity:  &activity,
	}
	xs.broadcastToClients(data)
}

func (xs *XBeeService) broadcastConnectionStatus(connected bool) {
	status := "disconnected"
	if connected {
		status = "connected"
	}

	data := LiveTelemetryData{
		Type:      "connection_status",
		Timestamp: time.Now(),
		Activity: &ActivityItem{
			Type:    status,
			Details: fmt.Sprintf("XBee %s", status),
		},
	}
	xs.broadcastToClients(data)
}

func (xs *XBeeService) broadcastToClients(data LiveTelemetryData) {
	xs.clientsMutex.RLock()
	defer xs.clientsMutex.RUnlock()

	for client := range xs.clients {
		err := client.WriteJSON(data)
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close()
			delete(xs.clients, client)
		}
	}
}

// WebSocket Handler
func (xs *XBeeService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := xs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	xs.clientsMutex.Lock()
	xs.clients[conn] = true
	xs.clientsMutex.Unlock()

	defer func() {
		xs.clientsMutex.Lock()
		delete(xs.clients, conn)
		xs.clientsMutex.Unlock()
	}()

	// Send initial stats and mission info
	if stats := xs.GetStats(); stats != nil {
		conn.WriteJSON(LiveTelemetryData{
			Type:  "stats_update",
			Stats: stats,
		})
	}

	if mission := xs.GetCurrentMission(); mission != nil {
		conn.WriteJSON(LiveTelemetryData{
			Type: "mission_info",
			Activity: &ActivityItem{
				Type:    "MISSION",
				Details: fmt.Sprintf("Current mission: %s", mission.Name),
			},
		})
	}

	// Handle incoming messages
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		response := xs.handleWebSocketMessage(msg)

		err = conn.WriteJSON(response)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

func (xs *XBeeService) handleWebSocketMessage(msg WSMessage) WSResponse {
	switch msg.Action {
	case "send_command":
		command, _ := msg.Data["command"].(string)
		dataStr, _ := msg.Data["data"].(string)

		var data []byte
		if dataStr != "" {
			data = []byte(dataStr)
		}

		err := xs.SendCommand(command, data)
		if err != nil {
			return WSResponse{
				Type:    "command_response",
				Success: false,
				Error:   err.Error(),
			}
		}

		return WSResponse{
			Type:    "command_response",
			Success: true,
		}

	case "start_mission":
		missionName, _ := msg.Data["name"].(string)
		if missionName == "" {
			missionName = fmt.Sprintf("Mission_%s", time.Now().Format("2006-01-02_15-04-05"))
		}

		err := xs.StartNewMission(missionName)
		if err != nil {
			return WSResponse{
				Type:    "mission_response",
				Success: false,
				Error:   err.Error(),
			}
		}

		return WSResponse{
			Type:    "mission_response",
			Success: true,
			Data:    xs.GetCurrentMission(),
		}

	case "get_stats":
		return WSResponse{
			Type:    "stats_response",
			Success: true,
			Data:    xs.GetStats(),
		}

	case "get_activity":
		limit := 50
		if l, ok := msg.Data["limit"].(float64); ok {
			limit = int(l)
		}

		return WSResponse{
			Type:    "activity_response",
			Success: true,
			Data:    xs.GetActivityLog(limit),
		}

	default:
		return WSResponse{
			Type:    "error",
			Success: false,
			Error:   fmt.Sprintf("Unknown action: %s", msg.Action),
		}
	}
}

// Database helper methods
func (xs *XBeeService) GetTelemetryByTimeRange(startTime, endTime int64) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := xs.database.Where("rtc_epoch BETWEEN ? AND ?", startTime, endTime).
		Order("rtc_epoch asc").
		Find(&telemetries).Error
	return telemetries, err
}

func (xs *XBeeService) GetLatestTelemetry(limit int) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := xs.database.Order("rtc_epoch desc").Limit(limit).Find(&telemetries).Error
	return telemetries, err
}

func (xs *XBeeService) GetTelemetryStatsByTimeRange(startTime, endTime int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get count
	var count int64
	if err := xs.database.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ?", startTime, endTime).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["total_packets"] = count

	// Get altitude range if data exists
	var maxAlt, minAlt, avgAlt float64
	if err := xs.database.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ? AND altitude IS NOT NULL", startTime, endTime).
		Select("COALESCE(MAX(altitude), 0) as max_alt, COALESCE(MIN(altitude), 0) as min_alt, COALESCE(AVG(altitude), 0) as avg_alt").
		Row().Scan(&maxAlt, &minAlt, &avgAlt); err != nil {
		// If no data, set zeros
		maxAlt, minAlt, avgAlt = 0, 0, 0
	}
	stats["max_altitude"] = maxAlt
	stats["min_altitude"] = minAlt
	stats["avg_altitude"] = avgAlt

	// Get temperature range if data exists
	var maxTemp, minTemp, avgTemp float64
	if err := xs.database.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ? AND temperature IS NOT NULL", startTime, endTime).
		Select("COALESCE(MAX(temperature), 0) as max_temp, COALESCE(MIN(temperature), 0) as min_temp, COALESCE(AVG(temperature), 0) as avg_temp").
		Row().Scan(&maxTemp, &minTemp, &avgTemp); err != nil {
		// If no data, set zeros
		maxTemp, minTemp, avgTemp = 0, 0, 0
	}
	stats["max_temperature"] = maxTemp
	stats["min_temperature"] = minTemp
	stats["avg_temperature"] = avgTemp

	return stats, nil
}

// Conversation History Methods
func (xs *XBeeService) GetConversationHistory(missionID string) ([]models.ConversationHistory, error) {
	var conversations []models.ConversationHistory
	err := xs.database.Where("mission_id = ?", missionID).
		Order("timestamp ASC").
		Find(&conversations).Error
	return conversations, err
}

func (xs *XBeeService) GetConversationHistoryPaginated(missionID string, limit, offset int) ([]models.ConversationHistory, error) {
	var conversations []models.ConversationHistory
	err := xs.database.Where("mission_id = ?", missionID).
		Order("timestamp ASC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error
	return conversations, err
}

func (xs *XBeeService) GetConversationHistoryCount(missionID string) (int64, error) {
	var count int64
	err := xs.database.Model(&models.ConversationHistory{}).
		Where("mission_id = ?", missionID).
		Count(&count).Error
	return count, err
}

// Utility Functions
func parseCSV(data string) []string {
	// Simple CSV parsing - you might want to use encoding/csv for production
	fields := make([]string, 0)
	current := ""
	inQuotes := false

	for _, char := range data {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				fields = append(fields, current)
				current = ""
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}

	if current != "" {
		fields = append(fields, current)
	}

	return fields
}
