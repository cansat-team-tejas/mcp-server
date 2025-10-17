package xbee

import (
	"fmt"
	"log"
	"time"
)

type FrameProcessor struct {
	frameHandler func(XBeeFrameData) error
	errorHandler func(error)
	writer       func([]byte) (int, error)
	frameIdCount byte
	buffer       []byte
	state        int // 0 = waiting for delimiter, 1 = reading length, 2 = reading frame
	expectedLen  int
	escapeNext   bool
}

const (
	FRAME_DELIMITER = 0x7E
	STATE_DELIMITER = 0
	STATE_LENGTH    = 1
	STATE_FRAME     = 2

	packetTypeTelemetry   = 0x0001
	packetTypeLog         = 0x0002
	packetTypeCmdResponse = 0x0003
)

func NewFrameProcessor() *FrameProcessor {
	return &FrameProcessor{
		buffer: make([]byte, 0, 256),
		state:  STATE_DELIMITER,
	}
}

// Set frame handler
func (fp *FrameProcessor) SetFrameHandler(handler func(XBeeFrameData) error) {
	fp.frameHandler = handler
}

// Set error handler
func (fp *FrameProcessor) SetErrorHandler(handler func(error)) {
	fp.errorHandler = handler
}

// SetWriter injects a transport layer used when transmitting frames out over serial.
func (fp *FrameProcessor) SetWriter(writer func([]byte) (int, error)) {
	fp.writer = writer
}

// Process incoming bytes with basic XBee frame parsing
func (fp *FrameProcessor) ProcessByte(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	for _, b := range data {
		switch fp.state {
		case STATE_DELIMITER:
			if b == FRAME_DELIMITER {
				fp.buffer = fp.buffer[:0] // Clear buffer
				fp.state = STATE_LENGTH
				fp.escapeNext = false
			}

		case STATE_LENGTH:
			if len(fp.buffer) == 0 {
				// First length byte (MSB)
				fp.buffer = append(fp.buffer, b)
			} else {
				// Second length byte (LSB)
				fp.buffer = append(fp.buffer, b)
				fp.expectedLen = int(fp.buffer[0])<<8 + int(fp.buffer[1])
				fp.buffer = fp.buffer[:0] // Clear buffer for frame data
				fp.state = STATE_FRAME
			}

		case STATE_FRAME:
			if fp.escapeNext {
				fp.buffer = append(fp.buffer, b^0x20)
				fp.escapeNext = false
			} else if b == 0x7D {
				fp.escapeNext = true
				continue
			} else {
				fp.buffer = append(fp.buffer, b)
			}

			if len(fp.buffer) >= fp.expectedLen+1 { // +1 for checksum
				// Process complete frame
				fp.processFrame(fp.buffer[:fp.expectedLen])
				fp.state = STATE_DELIMITER
				fp.escapeNext = false
			}
		}
	}

	return nil
}

// Process a complete XBee frame
func (fp *FrameProcessor) processFrame(frameData []byte) {
	if len(frameData) < 1 {
		return
	}

	frameType := frameData[0]
	frame := XBeeFrameData{
		Timestamp: time.Now(),
		FrameType: int(frameType),
	}

	switch frameType {
	case 0x88: // AT Response
		frame.Type = "AT_RESPONSE"
		if len(frameData) >= 4 {
			frame.FrameId = frameData[1]
			frame.Command = string(frameData[2:4])
			frame.Status = frameData[4]
			if len(frameData) > 5 {
				frame.Value = frameData[5:]
			}
		}

	case 0x90: // ZigBee RX
		frame.Type = "RX_PACKET"
		if len(frameData) >= 12 {
			// Extract 64-bit and 16-bit addresses
			frame.Remote64 = fmt.Sprintf("%016X",
				uint64(frameData[1])<<56|uint64(frameData[2])<<48|
					uint64(frameData[3])<<40|uint64(frameData[4])<<32|
					uint64(frameData[5])<<24|uint64(frameData[6])<<16|
					uint64(frameData[7])<<8|uint64(frameData[8]))
			frame.Remote16 = fmt.Sprintf("%04X", uint16(frameData[9])<<8|uint16(frameData[10]))

			if len(frameData) > 12 {
				frame.Data = string(frameData[12:])
			}
		}

	case 0x91: // Explicit ZigBee RX
		frame.Type = "EXPLICIT_RX_PACKET"
		if len(frameData) >= 18 {
			// Extract addresses
			frame.Remote64 = fmt.Sprintf("%016X",
				uint64(frameData[1])<<56|uint64(frameData[2])<<48|
					uint64(frameData[3])<<40|uint64(frameData[4])<<32|
					uint64(frameData[5])<<24|uint64(frameData[6])<<16|
					uint64(frameData[7])<<8|uint64(frameData[8]))
			frame.Remote16 = fmt.Sprintf("%04X", uint16(frameData[9])<<8|uint16(frameData[10]))

			// Extract explicit metadata
			frame.ExplicitMetadata = &ExplicitMetadata{
				SourceEndpoint:      frameData[11],
				DestinationEndpoint: frameData[12],
				ClusterId:           uint16(frameData[13])<<8 | uint16(frameData[14]),
				ProfileId:           uint16(frameData[15])<<8 | uint16(frameData[16]),
			}

			// Map cluster IDs to packet types
			switch frame.ExplicitMetadata.ClusterId {
			case packetTypeTelemetry:
				frame.PacketType = "TELEMETRY"
			case packetTypeLog:
				frame.PacketType = "LOG"
			case packetTypeCmdResponse:
				frame.PacketType = "CMD_RESPONSE"
			default:
				frame.PacketType = "UNKNOWN"
			}

			if len(frameData) > 18 {
				frame.Data = string(frameData[18:])
			}
		}

	default:
		frame.Type = "UNKNOWN"
		frame.Data = fmt.Sprintf("%X", frameData)
	}

	// Call frame handler if set
	if fp.frameHandler != nil {
		if err := fp.frameHandler(frame); err != nil {
			fp.handleError(fmt.Errorf("frame handler error: %v", err))
		}
	}
}

// Send XBee frame (simplified version)
func (fp *FrameProcessor) SendFrame(frameData map[string]interface{}) error {
	frameType, ok := frameData["type"].(string)
	if !ok {
		return fmt.Errorf("frame type not specified")
	}

	fp.frameIdCount++

	switch frameType {
	case "AT_COMMAND":
		command, _ := frameData["command"].(string)
		parameter, _ := frameData["commandParameter"].([]byte)
		return fp.sendATCommand(command, parameter)

	case "TX_REQUEST":
		data, _ := frameData["data"].([]byte)
		addr64, _ := frameData["destination64"].(uint64)
		addr16, _ := frameData["destination16"].(uint16)
		return fp.sendDataPacket(data, addr64, addr16)

	default:
		return fmt.Errorf("unsupported frame type: %s", frameType)
	}
}

// Build AT command frame
func (fp *FrameProcessor) sendATCommand(command string, parameter []byte) error {
	if len(command) != 2 {
		return fmt.Errorf("AT command must be 2 characters")
	}

	frame := []byte{0x08, fp.frameIdCount} // Frame type + Frame ID
	frame = append(frame, command...)      // AT Command
	frame = append(frame, parameter...)    // Parameter

	return fp.sendRawFrame(frame)
}

// Build TX request frame
func (fp *FrameProcessor) sendDataPacket(data []byte, addr64 uint64, addr16 uint16) error {
	frame := []byte{0x10, fp.frameIdCount} // Frame type + Frame ID

	// 64-bit address
	for i := 7; i >= 0; i-- {
		frame = append(frame, byte(addr64>>(i*8)))
	}

	// 16-bit address
	frame = append(frame, byte(addr16>>8), byte(addr16))

	// Broadcast radius and options
	frame = append(frame, 0x00, 0x00)

	// Data
	frame = append(frame, data...)

	return fp.sendRawFrame(frame)
}

// Send raw frame with delimiter, length, and checksum
func (fp *FrameProcessor) sendRawFrame(frameData []byte) error {
	// Calculate length
	length := len(frameData)

	// Calculate checksum
	checksum := byte(0xFF)
	for _, b := range frameData {
		checksum -= b
	}

	// Build complete frame
	packet := []byte{FRAME_DELIMITER}                      // Delimiter
	packet = append(packet, byte(length>>8), byte(length)) // Length
	packet = append(packet, frameData...)                  // Frame data
	packet = append(packet, checksum)                      // Checksum

	escaped := fp.escapeFrame(packet)

	log.Printf("Sending XBee frame: %X", escaped)

	if fp.writer != nil {
		if _, err := fp.writer(escaped); err != nil {
			fp.handleError(fmt.Errorf("failed to write XBee frame: %w", err))
			return err
		}
		return nil
	}

	return fmt.Errorf("no XBee writer configured")
}

func (fp *FrameProcessor) escapeFrame(packet []byte) []byte {
	if len(packet) == 0 {
		return packet
	}

	escaped := []byte{packet[0]}

	for i := 1; i < len(packet); i++ {
		b := packet[i]
		switch b {
		case 0x7E, 0x7D, 0x11, 0x13:
			escaped = append(escaped, 0x7D, b^0x20)
		default:
			escaped = append(escaped, b)
		}
	}

	return escaped
}

func (fp *FrameProcessor) handleError(err error) {
	if fp.errorHandler != nil {
		fp.errorHandler(err)
	} else {
		log.Printf("XBee frame processor error: %v", err)
	}
}
