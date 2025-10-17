package xbee

import (
	"fmt"
	"log"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type SerialManager struct {
	port             serial.Port
	isOpen           bool
	portPath         string
	config           SerialConfig
	dataHandle       func([]byte) error
	disconnectHandle func() // Callback for when connection is lost
}

func NewSerialManager() *SerialManager {
	return &SerialManager{
		isOpen: false,
	}
}

// List available serial ports
func (sm *SerialManager) ListPorts() ([]PortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to list ports: %v", err)
	}

	var portInfos []PortInfo
	for _, port := range ports {
		info := PortInfo{
			Name:         port.Name,
			Description:  port.Product,
			SerialNumber: port.SerialNumber,
			VendorID:     port.VID,
			ProductID:    port.PID,
		}
		portInfos = append(portInfos, info)
	}

	return portInfos, nil
}

// Open serial connection
func (sm *SerialManager) Open(portPath string, config SerialConfig) error {
	if sm.isOpen {
		sm.Close()
	}

	// Convert parity string to serial.Parity
	var parity serial.Parity
	switch config.Parity {
	case "even":
		parity = serial.EvenParity
	case "odd":
		parity = serial.OddParity
	case "mark":
		parity = serial.MarkParity
	case "space":
		parity = serial.SpaceParity
	default:
		parity = serial.NoParity
	}

	// Convert stop bits
	var stopBits serial.StopBits
	switch config.StopBits {
	case 2:
		stopBits = serial.TwoStopBits
	case 15: // 1.5 stop bits
		stopBits = serial.OnePointFiveStopBits
	default:
		stopBits = serial.OneStopBit
	}

	mode := &serial.Mode{
		BaudRate: config.BaudRate,
		Parity:   parity,
		DataBits: config.DataBits,
		StopBits: stopBits,
	}

	port, err := serial.Open(portPath, mode)
	if err != nil {
		return fmt.Errorf("failed to open port %s: %v", portPath, err)
	}

	sm.port = port
	sm.isOpen = true
	sm.portPath = portPath
	sm.config = config

	// Start reading in goroutine
	go sm.readLoop()

	return nil
}

// Close serial connection
func (sm *SerialManager) Close() error {
	if !sm.isOpen {
		return nil
	}

	sm.isOpen = false
	if sm.port != nil {
		err := sm.port.Close()
		sm.port = nil
		return err
	}
	return nil
}

// Write data to serial port
func (sm *SerialManager) Write(data []byte) (int, error) {
	if !sm.isOpen || sm.port == nil {
		return 0, fmt.Errorf("serial port not connected")
	}
	return sm.port.Write(data)
}

// Set data handler function
func (sm *SerialManager) SetDataHandler(handler func([]byte) error) {
	sm.dataHandle = handler
}

// Set disconnect handler function
func (sm *SerialManager) SetDisconnectHandler(handler func()) {
	sm.disconnectHandle = handler
}

// Read loop for continuous data reading
func (sm *SerialManager) readLoop() {
	buffer := make([]byte, 1024)
	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for sm.isOpen {
		n, err := sm.port.Read(buffer)
		if err != nil {
			consecutiveErrors++
			if sm.isOpen { // Only log if we're supposed to be open
				log.Printf("Serial read error (%d/%d): %v", consecutiveErrors, maxConsecutiveErrors, err)

				// Check if this might be a disconnection
				if consecutiveErrors >= maxConsecutiveErrors {
					log.Printf("XBee appears to be disconnected after %d consecutive errors", consecutiveErrors)
					sm.handleDisconnection()
					break
				}
			}
			continue
		}

		// Reset error counter on successful read
		consecutiveErrors = 0

		if n > 0 && sm.dataHandle != nil {
			// Process each byte individually for XBee frame parsing
			for i := 0; i < n; i++ {
				if err := sm.dataHandle(buffer[i : i+1]); err != nil {
					log.Printf("Data handler error: %v", err)
				}
			}
		}
	}
}

// Handle disconnection event
func (sm *SerialManager) handleDisconnection() {
	log.Println("Handling XBee disconnection...")
	sm.isOpen = false

	if sm.port != nil {
		sm.port.Close()
		sm.port = nil
	}

	// Notify the service about disconnection
	if sm.disconnectHandle != nil {
		go sm.disconnectHandle() // Run in goroutine to avoid blocking
	}
}

// Status information
func (sm *SerialManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"isOpen": sm.isOpen,
		"path":   sm.portPath,
		"config": sm.config,
	}
}

func (sm *SerialManager) IsOpen() bool {
	return sm.isOpen
}

// DetectXBeePort attempts to automatically detect an XBee device
// Returns the port path and info if found, empty string if not found
func (sm *SerialManager) DetectXBeePort() (string, PortInfo, error) {
	ports, err := sm.ListPorts()
	if err != nil {
		return "", PortInfo{}, err
	}

	// Common XBee vendor IDs and patterns
	xbeeIndicators := []struct {
		vendorID    string
		productID   string
		description string
	}{
		{"0403", "6001", "FT232"},  // FTDI-based XBee adapters
		{"0403", "6015", "FT231"},  // FTDI FT231X
		{"10C4", "EA60", "CP210x"}, // Silicon Labs CP210x
		{"067B", "2303", "PL2303"}, // Prolific PL2303
		{"", "", "XBee"},           // Generic XBee in description
		{"", "", "USB Serial"},     // Generic USB serial
		{"2341", "", "Arduino"},    // Arduino with XBee shield
	}

	for _, port := range ports {
		for _, indicator := range xbeeIndicators {
			matched := false

			// Check vendor ID match
			if indicator.vendorID != "" && port.VendorID == indicator.vendorID {
				if indicator.productID == "" || port.ProductID == indicator.productID {
					matched = true
				}
			}

			// Check description match
			if indicator.description != "" && port.Description != "" {
				if contains(port.Description, indicator.description) {
					matched = true
				}
			}

			if matched {
				log.Printf("Auto-detected XBee device: %s (%s)", port.Name, port.Description)
				return port.Name, port, nil
			}
		}
	}

	return "", PortInfo{}, fmt.Errorf("no XBee device detected")
}

// Helper function for case-insensitive string contains
func contains(s, substr string) bool {
	if s == "" || substr == "" {
		return false
	}
	// Simple substring search
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetPortPath returns the currently connected port path
func (sm *SerialManager) GetPortPath() string {
	return sm.portPath
}
