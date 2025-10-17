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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"goapp/internal/models"
	"goapp/internal/telemetry"
)

type wsClient interface {
	WriteJSON(v interface{}) error
	ReadJSON(v interface{}) error
	Close() error
}

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
	clients      map[wsClient]struct{}
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

	// Communication logs for commands and responses
	commStore *communicationStore

	// Database integration
	database   *gorm.DB
	dbPath     string
	missionDir string
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

type logPayload struct {
	Raw            string `json:"raw"`
	Timestamp      string `json:"timestamp,omitempty"`
	EncodedCommand string `json:"encoded_command,omitempty"`
}

const (
	defaultRemoteAddr64Hex  = "0013A20042367EBB"
	defaultRemoteAddr16Hex  = "FFFE"
	defaultRemoteAddr64Uint = 0x0013A20042367EBB
	defaultRemoteAddr16Uint = 0xFFFE
)

func NewXBeeService(dbPath, missionDir string) (*XBeeService, error) {
	// Initialize database using existing telemetry package
	db, err := telemetry.EnsureSchema(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	if missionDir == "" {
		missionDir = "missions"
	}
	missionDir = filepath.Clean(missionDir)

	service := &XBeeService{
		serialManager:  NewSerialManager(),
		frameProcessor: NewFrameProcessor(),
		database:       db,
		dbPath:         dbPath,
		missionDir:     missionDir,
		clients:        make(map[wsClient]struct{}),
		activityLog:    make([]ActivityItem, 0, 1000),
		commStore:      newCommunicationStore(10000),
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
	service.frameProcessor.SetWriter(service.serialManager.Write)

	// Set up serial manager handlers
	service.serialManager.SetDataHandler(service.frameProcessor.ProcessByte)
	service.serialManager.SetDisconnectHandler(service.handleDisconnection)

	// Start statistics update goroutine
	go service.updateStatsLoop()

	// Start auto-reconnect monitoring (enabled by default)
	service.StartAutoReconnect(true)

	// Attempt initial auto-connect
	go func() {
		time.Sleep(2 * time.Second) // Give the service time to fully initialize
		log.Println("Attempting initial XBee auto-connect...")
		response := service.AutoConnectXBee()
		if response.Success {
			log.Printf("Initial auto-connect successful: %s", response.Message)
		} else {
			log.Printf("Initial auto-connect failed: %s (will retry automatically)", response.Error)
		}
	}()

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
	missionDBPath := filepath.Join(xs.missionDir, fmt.Sprintf("%s.db", missionID))

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

// AutoDetectXBee attempts to automatically detect an XBee device
func (xs *XBeeService) AutoDetectXBee() Response {
	portPath, portInfo, err := xs.serialManager.DetectXBeePort()
	if err != nil {
		return Response{
			Success: false,
			Error:   fmt.Sprintf("Auto-detection failed: %v", err),
		}
	}

	return Response{
		Success: true,
		Message: fmt.Sprintf("Detected XBee on %s", portPath),
		Ports:   []PortInfo{portInfo},
	}
}

// AutoConnectXBee automatically detects and connects to an XBee device
func (xs *XBeeService) AutoConnectXBee() Response {
	// First detect the XBee
	portPath, portInfo, err := xs.serialManager.DetectXBeePort()
	if err != nil {
		xs.statsMutex.Lock()
		xs.stats.ConnectionStatus = "disconnected"
		xs.stats.LastUpdate = time.Now()
		xs.statsMutex.Unlock()

		return Response{
			Success: false,
			Error:   fmt.Sprintf("No XBee device detected: %v", err),
		}
	}

	// Use default XBee configuration
	config := SerialConfig{
		BaudRate: 115200, // Point-to-point CanSat link
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
	}

	// Attempt connection
	err = xs.serialManager.Open(portPath, config)
	if err != nil {
		xs.statsMutex.Lock()
		xs.stats.ConnectionStatus = "error"
		xs.stats.LastUpdate = time.Now()
		xs.statsMutex.Unlock()

		return Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to connect to %s: %v", portPath, err),
		}
	}

	// Update connection status
	xs.statsMutex.Lock()
	xs.stats.ConnectionStatus = "connected"
	xs.stats.ConnectionUptime = time.Now()
	xs.stats.LastUpdate = time.Now()
	xs.statsMutex.Unlock()

	message := fmt.Sprintf("Auto-connected to XBee on %s (%s)", portPath, portInfo.Description)
	xs.addActivity("CONNECTION", "AUTO_CONNECTED", message)
	xs.broadcastConnectionStatus(true)

	return Response{
		Success: true,
		Message: message,
		Ports:   []PortInfo{portInfo},
	}
}

// StartAutoReconnect starts a background goroutine that monitors the connection
// and automatically reconnects if the XBee is disconnected
func (xs *XBeeService) StartAutoReconnect(enabled bool) {
	if !enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Check if we're disconnected
			xs.statsMutex.RLock()
			status := xs.stats.ConnectionStatus
			xs.statsMutex.RUnlock()

			if status != "connected" && !xs.serialManager.IsOpen() {
				log.Println("XBee disconnected, attempting auto-reconnect...")

				response := xs.AutoConnectXBee()
				if response.Success {
					log.Printf("Auto-reconnect successful: %s", response.Message)
					xs.addActivity("CONNECTION", "AUTO_RECONNECT", response.Message)
				} else {
					log.Printf("Auto-reconnect failed: %s", response.Error)
				}
			}
		}
	}()
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

	// Point-to-point addressing for the CanSat radio link
	remoteAddr64 := defaultRemoteAddr64Hex
	remoteAddr16 := defaultRemoteAddr16Hex

	// Send AT command or data packet based on command type
	var err error
	var rawData []byte
	if len(command) == 2 { // AT Command
		rawData = append([]byte(command), data...)
		err = xs.frameProcessor.sendATCommand(command, data)
		xs.updateStats("command_sent", command)
	} else { // Data packet
		// For data packets, we might need to specify destination addresses
		// For now, use broadcast addresses
		rawData = append([]byte(command), data...)
		err = xs.frameProcessor.sendDataPacket(rawData, defaultRemoteAddr64Uint, defaultRemoteAddr16Uint)
		xs.updateStats("data_sent", command)
	}

	if err != nil {
		return err
	}

	// Log the command with communication log system
	xs.LogCommand(command, rawData, remoteAddr64, remoteAddr16)

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
	clusterID, profileID := xs.updateFrameStatistics(frame)

	if frame.PacketType == "TELEMETRY" {
		xs.processTelemetryFrame(frame)
	} else {
		xs.processNonTelemetryFrame(frame, clusterID, profileID)
	}

	xs.addActivity("FRAME_RECEIVED", frame.Type, fmt.Sprintf("Received %s frame", frame.Type))
	return nil
}

func (xs *XBeeService) updateFrameStatistics(frame XBeeFrameData) (uint16, uint16) {
	xs.statsMutex.Lock()
	defer xs.statsMutex.Unlock()

	xs.stats.PacketsReceived++
	xs.stats.LastUpdate = time.Now()
	xs.stats.LastDataReceived = time.Now()

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

	if frame.ExplicitMetadata == nil {
		return 0, 0
	}

	return frame.ExplicitMetadata.ClusterId, frame.ExplicitMetadata.ProfileId
}

func (xs *XBeeService) processTelemetryFrame(frame XBeeFrameData) {
	telemetry, err := xs.parseTelemetryData(frame.Data)
	if err != nil {
		log.Printf("Failed to parse telemetry: %v", err)
		xs.addActivity("ERROR", "PARSE_ERROR", fmt.Sprintf("Failed to parse telemetry: %v", err))
		return
	}

	if err := xs.database.Create(telemetry).Error; err != nil {
		log.Printf("Failed to store telemetry: %v", err)
		xs.addActivity("ERROR", "DB_ERROR", fmt.Sprintf("Failed to store telemetry: %v", err))
		return
	}

	xs.broadcastLiveTelemetry(telemetry)
	xs.storeConversation("telemetry", "received", frame.Data, "xbee", marshalMetadata(map[string]interface{}{
		"packet_type": frame.PacketType,
		"frame_type":  frame.Type,
	}))
}

func (xs *XBeeService) processNonTelemetryFrame(frame XBeeFrameData, clusterID, profileID uint16) {
	responseType := frame.PacketType
	if responseType == "" {
		responseType = frame.Type
	}

	parsedData := frame.Data
	metadata := map[string]interface{}{
		"packet_type": frame.PacketType,
		"frame_type":  frame.Type,
	}

	if frame.PacketType == "LOG" {
		if payload, ok := parseLogPayload(frame.Data); ok {
			if dataBytes, err := json.Marshal(payload); err == nil {
				parsedData = string(dataBytes)
			}
			metadata["log"] = payload
		}
	}

	xs.LogResponse(responseType, []byte(frame.Data), parsedData, frame.Remote64, frame.Remote16, clusterID, profileID)

	messageType := "response"
	if frame.PacketType == "LOG" {
		messageType = "log"
	}

	displayData := parsedData
	if displayData == "" {
		displayData = frame.Data
	}

	xs.storeConversation(messageType, "received", displayData, "xbee", marshalMetadata(metadata))
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

const telemetryFieldCount = 41

// Telemetry Parsing (CSV format expected)
func (xs *XBeeService) parseTelemetryData(data string) (*models.Telemetry, error) {
	fields := parseCSV(data)
	if len(fields) < telemetryFieldCount {
		return nil, fmt.Errorf("insufficient telemetry fields: got %d", len(fields))
	}

	trim := func(idx int) string {
		if idx >= len(fields) {
			return ""
		}
		return strings.TrimSpace(fields[idx])
	}

	parseFloat := func(idx int) *float64 {
		value := trim(idx)
		if value == "" {
			return nil
		}
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return &v
		}
		return nil
	}

	parseInt := func(idx int) *int {
		value := trim(idx)
		if value == "" {
			return nil
		}
		if v, err := strconv.Atoi(value); err == nil {
			return &v
		}
		return nil
	}

	parseInt64 := func(idx int) *int64 {
		value := trim(idx)
		if value == "" {
			return nil
		}
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return &v
		}
		return nil
	}

	telemetry := &models.Telemetry{}

	if val := trim(0); val != "" {
		telemetry.TeamID = &val
	}

	telemetry.MissionTimeS = parseFloat(1)
	telemetry.PacketCount = parseInt(2)
	telemetry.Altitude = parseFloat(3)
	telemetry.Pressure = parseFloat(4)
	telemetry.Temperature = parseFloat(5)
	telemetry.Voltage = parseFloat(6)

	if val := trim(7); val != "" {
		telemetry.GnssTime = &val
	}

	telemetry.Latitude = parseFloat(8)
	telemetry.Longitude = parseFloat(9)
	telemetry.GpsAltitude = parseFloat(10)
	telemetry.Satellites = parseInt(11)

	offset := 0
	if len(fields) >= 44 {
		telemetry.IrnssInView = parseInt(12)
		telemetry.IrnssUsed = parseInt(13)
		telemetry.IrnssMask = parseInt64(14)
		offset = 3
	}

	telemetry.AccelX = parseFloat(12 + offset)
	telemetry.AccelY = parseFloat(13 + offset)
	telemetry.AccelZ = parseFloat(14 + offset)
	telemetry.GyroSpinRate = parseFloat(15 + offset)
	telemetry.FlightState = parseInt(16 + offset)
	telemetry.GyroX = parseFloat(17 + offset)
	telemetry.GyroY = parseFloat(18 + offset)
	telemetry.GyroZ = parseFloat(19 + offset)
	telemetry.Roll = parseFloat(20 + offset)
	telemetry.Pitch = parseFloat(21 + offset)
	telemetry.Yaw = parseFloat(22 + offset)
	telemetry.MagX = parseFloat(23 + offset)
	telemetry.MagY = parseFloat(24 + offset)
	telemetry.MagZ = parseFloat(25 + offset)
	telemetry.Humidity = parseFloat(26 + offset)
	telemetry.Current = parseFloat(27 + offset)
	telemetry.Power = parseFloat(28 + offset)
	telemetry.BaroAltitude = parseFloat(29 + offset)
	telemetry.AirQualityRaw = parseInt(30 + offset)
	telemetry.AqEthanolPpm = parseFloat(31 + offset)
	telemetry.McuTempC = parseFloat(32 + offset)
	telemetry.RssiDbm = parseInt(33 + offset)

	if val := trim(34 + offset); val != "" {
		telemetry.HealthFlags = &val
	}

	if rtc := trim(35 + offset); rtc != "" {
		if parsed, err := strconv.ParseInt(rtc, 10, 64); err == nil {
			v := int(parsed)
			telemetry.RtcEpoch = &v
		}
	}

	telemetry.RwSpeedPct = parseInt(36 + offset)
	telemetry.RwSaturated = parseInt(37 + offset)
	telemetry.YawRateTarget = parseFloat(38 + offset)
	telemetry.PidOutput = parseFloat(39 + offset)

	if val := trim(40 + offset); val != "" {
		telemetry.CmdEcho = &val
	}

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
	clients := make([]wsClient, 0, len(xs.clients))
	for client := range xs.clients {
		clients = append(clients, client)
	}
	xs.clientsMutex.RUnlock()

	for _, client := range clients {
		if err := client.WriteJSON(data); err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close()
			xs.clientsMutex.Lock()
			delete(xs.clients, client)
			xs.clientsMutex.Unlock()
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

	xs.serveWebSocketConnection(conn)
}

func (xs *XBeeService) ServeWebSocketConn(conn wsClient) {
	if conn == nil {
		return
	}

	xs.serveWebSocketConnection(conn)
}

func (xs *XBeeService) serveWebSocketConnection(conn wsClient) {
	defer conn.Close()

	xs.clientsMutex.Lock()
	xs.clients[conn] = struct{}{}
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
func parseLogPayload(data string) (logPayload, bool) {
	payload := logPayload{Raw: data}

	trimmed := strings.TrimSpace(data)
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return payload, false
	}

	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return payload, false
	}

	parts := strings.Fields(inner)
	if len(parts) < 2 {
		return payload, false
	}

	payload.Timestamp = parts[0]
	payload.EncodedCommand = strings.Join(parts[1:], " ")
	return payload, true
}

func marshalMetadata(meta map[string]interface{}) string {
	if meta == nil {
		return "{}"
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(data)
}

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

	fields = append(fields, current)

	return fields
}

// Communication Log Methods

// LogCommand records a command sent to the XBee/CanSat
func (xs *XBeeService) LogCommand(command string, rawData []byte, remoteAddr64, remoteAddr16 string) CommandLog {
	missionID := xs.currentMissionID()
	entry := xs.commStore.recordCommand(command, missionID, rawData, remoteAddr64, remoteAddr16)

	log.Printf("Logged command: %s", command)
	return entry
}

// LogResponse records a response received from XBee/CanSat
func (xs *XBeeService) LogResponse(responseType string, rawData []byte, parsedData string, remoteAddr64, remoteAddr16 string, clusterID, profileID uint16) ResponseLog {
	missionID := xs.currentMissionID()
	entry := xs.commStore.recordResponse(responseType, missionID, rawData, parsedData, remoteAddr64, remoteAddr16, clusterID, profileID)

	log.Printf("Logged response: %s (Cluster: 0x%04X)", responseType, clusterID)
	return entry
}

func (xs *XBeeService) currentMissionID() string {
	xs.missionMutex.RLock()
	defer xs.missionMutex.RUnlock()

	if xs.currentMission != nil && xs.currentMission.IsActive {
		return xs.currentMission.ID
	}

	return ""
}

// GetCommunicationLogs returns a snapshot of all communication logs.
func (xs *XBeeService) GetCommunicationLogs() CommunicationSnapshot {
	return xs.commStore.snapshot()
}

// GetCommandLogs returns only the command history.
func (xs *XBeeService) GetCommandLogs() []CommandLog {
	return xs.commStore.latestCommands()
}

// GetResponseLogs returns only the response history.
func (xs *XBeeService) GetResponseLogs() []ResponseLog {
	return xs.commStore.latestResponses()
}
