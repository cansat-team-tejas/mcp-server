package xbee

import (
	"time"
)

// Response structures matching your Electron IPC responses
type Response struct {
	Success bool       `json:"success"`
	Error   string     `json:"error,omitempty"`
	Ports   []PortInfo `json:"ports,omitempty"`
}

type PortInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	HardwareID   string `json:"hardwareId"`
	VendorID     string `json:"vendorId"`
	ProductID    string `json:"productId"`
	SerialNumber string `json:"serialNumber"`
	LocationID   string `json:"locationId"`
	Manufacturer string `json:"manufacturer"`
}

type SerialConfig struct {
	BaudRate int    `json:"baudRate"`
	DataBits int    `json:"dataBits"`
	StopBits int    `json:"stopBits"`
	Parity   string `json:"parity"`
}

type XBeeFrameData struct {
	Type             string            `json:"type"`
	FrameType        int               `json:"frameType"`
	FrameId          byte              `json:"frameId"`
	Command          string            `json:"command,omitempty"`
	Status           byte              `json:"status,omitempty"`
	Value            interface{}       `json:"value,omitempty"`
	Data             string            `json:"data,omitempty"`
	PacketType       string            `json:"packetType,omitempty"`
	ExplicitMetadata *ExplicitMetadata `json:"explicitMetadata,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
	Remote64         string            `json:"remote64,omitempty"`
	Remote16         string            `json:"remote16,omitempty"`
}

type ExplicitMetadata struct {
	SourceEndpoint      byte   `json:"sourceEndpoint"`
	DestinationEndpoint byte   `json:"destinationEndpoint"`
	ClusterId           uint16 `json:"clusterId"`
	ProfileId           uint16 `json:"profileId"`
}

// Statistics and monitoring types
type ConnectionStats struct {
	PacketsReceived int       `json:"packetsReceived"`
	PacketsSent     int       `json:"packetsSent"`
	ErrorsCount     int       `json:"errorsCount"`
	ConnectedAt     time.Time `json:"connectedAt"`
	LastActivity    time.Time `json:"lastActivity"`
}

type FrameStats struct {
	TelemetryCount   int `json:"telemetryCount"`
	CommandEchoCount int `json:"commandEchoCount"`
	LogEntryCount    int `json:"logEntryCount"`
	UnknownCount     int `json:"unknownCount"`
}

type ActivityItem struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "FRAME_RECEIVED", "FRAME_SENT", "CONNECTION", "ERROR"
	FrameType string    `json:"frameType,omitempty"`
	Details   string    `json:"details,omitempty"`
}

// WebSocket message types
type WSMessage struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

type WSResponse struct {
	Type    string      `json:"type"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
