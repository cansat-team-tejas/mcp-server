# CanSat Mission Control Protocol (MCP) Server

This Go application provides a comprehensive REST API for CanSat telemetry data management, XBee serial communication, and AI-assisted querying. It uses [Fiber](https://gofiber.io/) as the web framework, GORM for database operations, and integrates with Ollama for local AI-powered responses.

## Features

- **XBee Serial Communication**: Full XBee integration with frame processing, command sending, and live telemetry streaming
- **Hands-Free XBee Connection**: Automatic detect/connect/reconnect cycle with no GUI action required
- **Mission Management**: Automatic mission creation with START command detection and per-mission databases
- **Live Data Streaming**: WebSocket endpoints for real-time telemetry and status updates
- **Conversation History**: Complete tracking of all mission communications (commands, telemetry, responses)
- **Multi-database support**: Each mission creates its own SQLite database file
- **AI-powered Q&A**: Natural language queries about telemetry data with automatic SQL generation using local Ollama
- **🤖 AI Instructions Configuration**: External TOML-based AI instruction management with context-aware responses
- **Command detection**: Recognizes GS (Ground Station) commands and provides formatted responses
- **Statistics & Analytics**: Real-time packet rates, connection status, and mission analytics
- **Schema auto-migration**: Automatically creates tables using GORM migrations

## New: AI Instructions System

🎉 **AI instructions are now externalized to `ai_instructions.toml`** for easy customization without code changes!

- **Context-Aware AI**: Specialized responses for telemetry analysis, mission planning, and error diagnosis
- **Hot Reload**: Update AI behavior without restarting the service
- **Configurable Settings**: Control response length, technical level, and safety priorities
- **Template System**: Reusable prompt templates with parameter substitution

📖 **See [AI_INSTRUCTIONS.md](./AI_INSTRUCTIONS.md) for complete documentation.**

## Prerequisites

- Go 1.22+
- Ollama installed and running locally with gemma3:4b model
- XBee radio module (for hardware communication)

## Configuration

Set the following environment variables before running the server:

| Variable      | Description            | Default                  |
| ------------- | ---------------------- | ------------------------ |
| `PORT`        | HTTP port to listen on | `8000`                   |
| `LLM_API_URL` | Ollama API endpoint    | `http://localhost:11434` |
| `LLM_MODEL`   | Ollama model name      | `gemma3:4b`              |

## Running

```powershell
cd mcp-server
go mod tidy
$env:CGO_ENABLED=0  # CGO-free build for better compatibility
go run ./cmd
```

### Test AI Instructions

```powershell
$env:CGO_ENABLED=0
go run ./cmd/demo  # Demonstrates AI instructions configuration
```

The server listens on `http://localhost:8000` by default.

## GUI Integration Quick Reference

The GUI can rely on the MCP server for everything from database access to XBee connectivity. Key endpoints:

📚 **Need the exact contracts?** Check the dedicated [GUI_API_REFERENCE.md](./GUI_API_REFERENCE.md).

- `GET /api/xbee/status` – live connection + mission status (auto-connected on startup)
- `GET /api/xbee/health` – connection health metrics (latency, last packet, uptime)
- `GET /api/xbee/logs` – combined command/response history for mission review
- `GET /api/xbee/logs/commands` – ground commands that were transmitted
- `GET /api/xbee/logs/responses` – frames received from the CanSat (excluding telemetry)
- `WS /api/xbee/ws` – live telemetry, activity, and command streaming channel
- Link configuration: 115200 baud, point-to-point, remote address `0013A20042367EBB`
- `POST /ask` – natural-language questions about telemetry + command detection
- `POST /data` – raw telemetry pull for dashboards

## API Endpoints

All endpoints expect JSON payloads and return JSON responses. Each request must include a `filename` parameter. Filenames are always resolved inside the configured `missions` directory.

### 1. Ask Questions

**POST** `/ask`

Processes natural language questions about telemetry data, generates SQL queries, executes them, and provides AI-powered conversational responses. Supports GS command detection.

If the specified database file doesn't exist, falls back to answering based on static context from a `.txt` file (e.g., `mission1.txt` for `mission1.db`, or `context.txt` as fallback).

**Request:**

```json
{
  "question": "What is the average altitude?",
  "filename": "mission1.db"
}
```

**Response:**

```json
{
  "answer": {
    "content": "The average altitude across all telemetry points is 1250.5 meters..."
  },
  "command": "ALT"
}
```

### 2. Get Telemetry Data

**POST** `/data`

Returns all telemetry data points from the specified database.

**Request:**

```json
{
  "filename": "mission1.db"
}
```

**Response:**

```json
[
  {
    "id": 1,
    "TEAM_ID": "TEJAS",
    "mission_time_s": 120.5,
    "altitude": 1250.5,
    "temperature": 25.3
    // ... all telemetry fields
  }
]
```

## AI Chat API Endpoints

The AI module provides intelligent chat functionality with support for both traditional request-response and real-time streaming responses using Ollama local LLM.

### Chat Endpoints

#### POST `/api/chat`

Processes chat messages with AI. Supports both streaming and non-streaming modes.

**Request:**

```json
{
  "messages": [
    {
      "role": "user",
      "content": "What is the optimal altitude for CanSat deployment?"
    }
  ],
  "stream": false
}
```

**Response (Non-streaming):**

```json
{
  "success": true,
  "message": "The optimal altitude for CanSat deployment typically ranges from 700-1000 meters, depending on mission requirements..."
}
```

**Response (Streaming via HTTP SSE):**
When `stream: true` is set, the response uses Server-Sent Events format:

```
data: {"type":"chunk","content":"The optimal","done":false}
data: {"type":"chunk","content":" altitude for","done":false}
data: {"type":"chunk","content":" CanSat deployment","done":false}
data: {"type":"done","done":true}
```

#### GET `/api/chat/ws`

WebSocket endpoint for real-time chat streaming with bidirectional communication.

**WebSocket Message Format:**

**Client → Server:**

```json
{
  "messages": [
    {
      "role": "user",
      "content": "Explain CanSat telemetry data structure"
    }
  ]
}
```

**Server → Client (Streaming chunks):**

```json
{
  "type": "chunk",
  "content": "CanSat telemetry typically includes",
  "done": false
}
```

**Server → Client (Completion):**

```json
{
  "type": "done",
  "done": true
}
```

**Server → Client (Error):**

```json
{
  "type": "error",
  "error": "Failed to process request",
  "done": true
}
```

#### GET `/api/chat/health`

Health check for AI service availability.

**Response:**

```json
{
  "success": true,
  "status": "AI service is running",
  "model": "llama3.1:8b"
}
```

### Chat Features

- **Real-time Streaming**: Get AI responses as they're generated
- **Multi-turn Conversations**: Maintain context across multiple messages
- **Local Processing**: Uses Ollama for privacy and reliability
- **WebSocket Support**: Bidirectional real-time communication
- **Error Handling**: Comprehensive error reporting and timeout management
- **Model Flexibility**: Configurable AI model selection

## XBee API Endpoints

The XBee module provides comprehensive serial communication capabilities for CanSat missions, including real-time telemetry streaming, command transmission, and mission management.

### Connection Monitoring (Automatic)

The MCP server automatically scans available serial ports on startup, connects to the detected XBee, and continually retries if the device is unplugged or reboots. The GUI **does not** need to expose manual connect/disconnect controls—just surface status and health data from the endpoints below.

#### GET `/api/xbee/status`

Returns current connection state, mission info, and latest statistics from the auto-managed link.

**Response:**

```json
{
  "success": true,
  "connection": {
    "isOpen": true,
    "port": "COM3",
    "config": {
      "baudRate": 115200,
      "dataBits": 8,
      "stopBits": 1,
      "parity": "none"
    }
  },
  "mission": {
    "id": "mission_1696752000",
    "name": "Mission_2023-10-08_10-00-00",
    "startTime": "2023-10-08T10:00:00Z",
    "isActive": true,
    "dbPath": "missions/mission_1696752000.db"
  },
  "stats": {
    "packetRate": 1.5,
    "packetsReceived": 150,
    "packetsSent": 5,
    "lastUpdate": "2023-10-08T10:30:00Z",
    "startTime": "2023-10-08T10:00:00Z",
    "lastCommand": "START",
    "lastCommandEcho": "START_ACK",
    "connectionStatus": "connected",
    "lastDataReceived": "2023-10-08T10:30:00Z",
    "connectionUptime": "2023-10-08T10:00:00Z"
  },
  "connection_health": {
    "is_connected": true,
    "connection_status": "connected",
    "time_since_last_data_ms": 1500,
    "last_data_received": "2023-10-08T10:30:00Z",
    "connection_uptime": "2023-10-08T10:00:00Z",
    "packets_received": 150,
    "packets_sent": 5,
    "health_status": "good",
    "health_reason": "Receiving data regularly"
  }
}
```

#### GET `/api/xbee/health`

Provides connection health diagnostics sourced from the auto-connect service.

**Response:**

```json
{
  "success": true,
  "health": {
    "is_connected": true,
    "connection_status": "connected",
    "time_since_last_data_ms": 1500,
    "last_data_received": "2023-10-08T10:30:00Z",
    "connection_uptime": "2023-10-08T10:00:00Z",
    "packets_received": 150,
    "packets_sent": 5,
    "health_status": "good",
    "health_reason": "Receiving data regularly"
  }
}
```

**Health Status Values:**

- `"good"`: Connection is healthy, receiving data regularly
- `"warning"`: No data received for >10 seconds but <30 seconds
- `"poor"`: No data received for >30 seconds
- `"disconnected"`: XBee is not connected

### Command & Data Transmission

#### POST `/api/xbee/command`

Sends commands to XBee module. Special handling for "START" command which creates new missions.

**Request:**

```json
{
  "command": "START",
  "data": ""
}
```

**Response:**

```json
{
  "success": true
}
```

### Communication Logs

The server captures every command sent to the XBee and every non-telemetry frame received, retaining the 10,000 most recent events in memory. Use these endpoints for GUI displays or mission debugging.

#### GET `/api/xbee/logs`

Returns combined command and response history with timestamps and mission correlation.

**Response:**

```json
{
  "success": true,
  "logs": [
    {
      "type": "command",
      "timestamp": "2023-10-08T10:05:00Z",
      "command": "START"
    },
    {
      "type": "response",
      "timestamp": "2023-10-08T10:05:01Z",
      "response": "START_ACK"
    }
  ]
}
```

#### GET `/api/xbee/logs/commands`

Returns only the commands that were transmitted to the spacecraft.

#### GET `/api/xbee/logs/responses`

Returns only the non-telemetry frames received from the spacecraft.

### Mission Management

#### POST `/api/xbee/mission/start`

Manually starts a new mission with custom name.

**Request:**

```json
{
  "name": "Custom Mission Name"
}
```

**Response:**

```json
{
  "success": true,
  "mission": {
    "id": "mission_1696752000",
    "name": "Custom Mission Name",
    "startTime": "2023-10-08T10:00:00Z",
    "isActive": true,
    "dbPath": "missions/mission_1696752000.db"
  }
}
```

#### GET `/api/xbee/mission`

Gets current active mission information.

**Response:**

```json
{
  "success": true,
  "mission": {
    "id": "mission_1696752000",
    "name": "Mission_2023-10-08_10-00-00",
    "startTime": "2023-10-08T10:00:00Z",
    "isActive": true,
    "dbPath": "missions/mission_1696752000.db"
  }
}
```

### Data Retrieval

#### GET `/api/xbee/stats`

Gets real-time statistics and performance metrics.

**Response:**

```json
{
  "success": true,
  "stats": {
    "packetRate": 1.5,
    "packetsReceived": 150,
    "packetsSent": 5,
    "lastUpdate": "2023-10-08T10:30:00Z",
    "startTime": "2023-10-08T10:00:00Z",
    "lastCommand": "START",
    "lastCommandEcho": "START_ACK",
    "frameStats": {
      "telemetryCount": 140,
      "logEntryCount": 5,
      "commandEchoCount": 5,
      "unknownCount": 0
    }
  }
}
```

#### GET `/api/xbee/telemetry`

Gets telemetry data with optional filtering.

**Query Parameters:**

- `start_time`: Unix timestamp for start range
- `end_time`: Unix timestamp for end range
- `limit`: Maximum number of records (default: 100)

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "TEAM_ID": "TEAM_001",
      "mission_time_s": 120.5,
      "packet_count": 1,
      "altitude": 1025.3,
      "temperature": 15.2,
      "rtc_epoch": 1696752060
    }
  ]
}
```

#### GET `/api/xbee/telemetry/stats`

Gets statistical analysis of telemetry data.

**Query Parameters:**

- `start_time`: Unix timestamp for start range
- `end_time`: Unix timestamp for end range

**Response:**

```json
{
  "success": true,
  "stats": {
    "total_packets": 150,
    "max_altitude": 1200.5,
    "min_altitude": 980.2,
    "avg_altitude": 1100.3,
    "max_temperature": 25.1,
    "min_temperature": -5.3,
    "avg_temperature": 12.4
  }
}
```

#### GET `/api/xbee/activity`

Gets recent activity log entries.

**Query Parameters:**

- `limit`: Maximum number of entries (default: 50)

**Response:**

```json
{
  "success": true,
  "activity": [
    {
      "timestamp": "2023-10-08T10:30:00Z",
      "type": "FRAME_RECEIVED",
      "action": "TELEMETRY",
      "details": "Received TELEMETRY frame"
    }
  ]
}
```

### Conversation History

#### GET `/api/xbee/conversation/:missionId`

Gets paginated conversation history for a specific mission.

**Query Parameters:**

- `limit`: Maximum number of messages (default: 100)
- `offset`: Number of messages to skip (default: 0)

**Response:**

```json
{
  "success": true,
  "conversations": [
    {
      "id": 1,
      "missionId": "mission_1696752000",
      "timestamp": "2023-10-08T10:30:00Z",
      "messageType": "command",
      "direction": "sent",
      "content": "START",
      "source": "gui",
      "metadata": "{\"data\": \"\"}"
    },
    {
      "id": 2,
      "missionId": "mission_1696752000",
      "timestamp": "2023-10-08T10:30:01Z",
      "messageType": "telemetry",
      "direction": "received",
      "content": "TEAM_001,120.5,1,1025.3,15.2,...",
      "source": "xbee",
      "metadata": "{\"packet_type\": \"TELEMETRY\"}"
    }
  ],
  "total_count": 150,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

#### GET `/api/xbee/conversation/:missionId/all`

Gets complete conversation history for a mission.

**Response:**

```json
{
  "success": true,
  "conversations": [
    // Array of all conversation messages
  ],
  "total_count": 150
}
```

### WebSocket Live Streaming

Connect to `/api/xbee/ws` for real-time updates:

**Message Types:**

- `telemetry`: Live telemetry data
- `stats`: Updated statistics
- `activity`: New activity log entries
- `connection_status`: Connection state changes

**Example WebSocket Message:**

```json
{
  "type": "telemetry",
  "timestamp": "2023-10-08T10:30:00Z",
  "data": {
    "TEAM_ID": "TEAM_001",
    "mission_time_s": 120.5,
    "altitude": 1025.3,
    "temperature": 15.2
  }
}
```

## Data Types

### Telemetry Fields

All telemetry fields are optional (nullable) in the database:

| Field                           | Type    | Description                  |
| ------------------------------- | ------- | ---------------------------- |
| `TEAM_ID`                       | string  | Team identifier              |
| `mission_time_s`                | float64 | Mission time in seconds      |
| `packet_count`                  | int     | Packet sequence number       |
| `altitude`                      | float64 | Altitude in meters           |
| `pressure`                      | float64 | Atmospheric pressure         |
| `temperature`                   | float64 | Temperature in Celsius       |
| `voltage`                       | float64 | Battery voltage              |
| `gnss_time`                     | string  | GNSS timestamp               |
| `latitude`                      | float64 | GPS latitude                 |
| `longitude`                     | float64 | GPS longitude                |
| `gps_altitude`                  | float64 | GPS altitude                 |
| `satellites`                    | int     | Number of GPS satellites     |
| `accel_x`, `accel_y`, `accel_z` | float64 | Accelerometer readings       |
| `gyro_spin_rate`                | float64 | Gyroscope spin rate          |
| `flight_state`                  | int     | Flight state code            |
| `gyro_x`, `gyro_y`, `gyro_z`    | float64 | Gyroscope readings           |
| `roll`, `pitch`, `yaw`          | float64 | Orientation angles           |
| `mag_x`, `mag_y`, `mag_z`       | float64 | Magnetometer readings        |
| `humidity`                      | float64 | Humidity percentage          |
| `current`                       | float64 | Current draw                 |
| `power`                         | float64 | Power consumption            |
| `baro_altitude`                 | float64 | Barometric altitude          |
| `air_quality_raw`               | int     | Raw air quality sensor value |
| `aq_ethanol_ppm`                | float64 | Ethanol concentration in ppm |
| `mcu_temp_c`                    | float64 | MCU temperature              |
| `rssi_dbm`                      | int     | Signal strength in dBm       |
| `health_flags`                  | string  | Health status flags          |
| `rtc_epoch`                     | int     | Real-time clock epoch        |
| `cmd_echo`                      | string  | Command echo                 |

### Conversation History Fields

| Field         | Type      | Description                                              |
| ------------- | --------- | -------------------------------------------------------- |
| `id`          | uint      | Auto-generated primary key                               |
| `missionId`   | string    | Mission identifier (indexed)                             |
| `timestamp`   | time.Time | When the message occurred                                |
| `messageType` | string    | Type: "command", "telemetry", "response", "error", "log" |
| `direction`   | string    | Direction: "sent" or "received"                          |
| `content`     | string    | The actual message content                               |
| `source`      | string    | Source: "gui", "xbee", or "system"                       |
| `metadata`    | string    | Additional context as JSON string                        |

### Mission Management

The system automatically creates new missions when:

1. A "START" command is sent via `/api/xbee/command`
2. A mission is manually started via `/api/xbee/mission/start`

Each mission:

- Gets a unique ID based on Unix timestamp
- Creates its own SQLite database file in `missions/` directory
- Tracks all conversation history (commands, telemetry, responses, errors)
- Stores telemetry data separately per mission
- Maintains real-time statistics and activity logs

### XBee Frame Processing

The system handles multiple XBee frame types:

- **TELEMETRY**: CSV-formatted sensor data automatically parsed and stored
- **LOG**: Log entries from the CanSat system
- **CMD_RESPONSE**: Command acknowledgments and responses
- **Unknown**: Other frame types logged for debugging

All frames are automatically logged in the conversation history with appropriate metadata.

## Notes

- **Mission Databases**: Each mission automatically creates its own SQLite database file
- **Real-time Communication**: XBee integration provides live bidirectional communication with CanSat
- **Connection Monitoring**: Automatic XBee disconnection detection with health status tracking
- **AI Chat Streaming**: Real-time AI responses via WebSocket and Server-Sent Events
- **Conversation Tracking**: Complete audit trail of all mission communications stored automatically
- **Thread Safety**: All database operations and WebSocket connections are thread-safe
- **Auto-Migration**: Telemetry (`telemetry`) and mission conversation (`conversation_histories`) tables are created/renamed automatically when first accessed
- **Local AI**: Uses Ollama for local AI processing without external API dependencies
- **Multi-protocol Support**: HTTP REST, WebSocket, and SSE for different client needs
- **Frame Processing**: Robust XBee frame parsing with error handling and logging
- **Live Streaming**: WebSocket support for real-time GUI updates
- **Connection Health**: Real-time monitoring of XBee connection status and data flow
- **Command Detection**: Special handling for mission control commands like "START"
- **Statistics**: Real-time packet rate monitoring and mission analytics
- **Error Handling**: Comprehensive error logging with detailed HTTP status responses

## Troubleshooting

### CGO Build Issues

If encountering CGO compilation errors:

```bash
$env:CGO_ENABLED=0
go build -o mcp-server.exe ./cmd
```

### XBee Connection Issues

- Verify XBee module is properly connected
- Check COM port availability and permissions
- Ensure correct baud rate configuration (115200 for the current CanSat link)
- Verify XBee firmware supports frame-based communication
- **Connection Monitoring**: Use `/api/xbee/health` to check connection status
- **Automatic Detection**: System detects disconnections after 5 consecutive read errors
- **Health Status**: Monitor `time_since_last_data_ms` for connection quality

### XBee Disconnection Detection

- **Automatic**: Detects connection loss through consecutive read errors
- **Real-time Alerts**: WebSocket broadcasts notify clients of disconnections
- **Health Monitoring**: `/api/xbee/health` provides detailed connection diagnostics
- **Conversation Logging**: All disconnection events are logged in conversation history

### Ollama Configuration

- Ensure Ollama is running: `ollama serve`
- Verify model is available: `ollama list`
- Pull required model: `ollama pull llama3.1:8b`
