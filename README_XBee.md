# MCP Server with XBee Integration

A comprehensive Go server that combines Model Context Protocol (MCP) functionality with XBee telemetry data handling for CanSat missions.

## Features

### 🚀 Core Functionality

- **Local LLM Integration**: Uses Ollama for AI-powered SQL query generation
- **XBee Serial Communication**: Full XBee protocol support for telemetry data
- **Mission Management**: Automatic database creation for new missions
- **Live Data Streaming**: Real-time telemetry data via WebSocket
- **RESTful API**: Complete HTTP API for all functionality

### 📡 XBee Features

- Serial port discovery and connection
- XBee frame parsing and processing
- Command transmission to remote XBee devices
- Real-time statistics tracking
- Activity logging

### 📊 Data Management

- Automatic telemetry data storage
- Mission-based database organization
- Statistical analysis and reporting
- Time-range queries

## Quick Start

### Prerequisites

- Go 1.21 or later
- Ollama (for local LLM)
- XBee device (for telemetry)

### Installation

```bash
# Clone the repository
git clone https://github.com/cansat-team-tejas/mcp-server.git
cd mcp-server

# Build the application
go build -o mcp-server.exe ./cmd

# Run the server
./mcp-server.exe
```

### Configuration

Configure via environment variables:

```bash
# Server Configuration
export PORT=8000
export DB_PATH="telemetry.db"

# LLM Configuration
export LLM_ENDPOINT="http://localhost:11434"
export LLM_MODEL="llama3.1:8b"
export LLM_TOKEN=""  # Optional

# XBee Configuration
export XBEE_BAUD_RATE=9600
export XBEE_DATA_BITS=8
export XBEE_STOP_BITS=1
export XBEE_PARITY="none"
export MISSION_DIR="missions"
```

## API Reference

### Base URL

```
http://localhost:8000
```

### XBee Endpoints

#### Connection Management

```http
GET /api/xbee/ports
```

List available serial ports

```http
POST /api/xbee/connect
Content-Type: application/json

{
  "port": "COM3",
  "config": {
    "baudRate": 9600,
    "dataBits": 8,
    "stopBits": 1,
    "parity": "none"
  }
}
```

```http
POST /api/xbee/disconnect
```

```http
GET /api/xbee/status
```

Get connection status, mission info, and current statistics

#### Command Transmission

```http
POST /api/xbee/command
Content-Type: application/json

{
  "command": "START",
  "data": "optional command data"
}
```

Special commands:

- `"START"`: Automatically creates a new mission and database

#### Mission Management

```http
POST /api/xbee/mission/start
Content-Type: application/json

{
  "name": "Mission Apollo 13"
}
```

```http
GET /api/xbee/mission
```

#### Data Retrieval

```http
GET /api/xbee/stats
```

Current statistics including:

- Packet rate (Hz)
- Packets received/sent
- Last command/command echo
- Frame statistics

```http
GET /api/xbee/telemetry?limit=100&start_time=1633024800&end_time=1633111200
```

```http
GET /api/xbee/telemetry/stats?start_time=1633024800&end_time=1633111200
```

```http
GET /api/xbee/activity?limit=50
```

### Live Data Streaming

#### WebSocket Connection

```javascript
const ws = new WebSocket("ws://localhost:8000/api/xbee/ws");

ws.onmessage = function (event) {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case "live_telemetry":
      // Real-time telemetry data
      console.log("Telemetry:", data.data);
      break;

    case "stats_update":
      // Updated statistics
      console.log("Stats:", data.stats);
      break;

    case "activity":
      // Activity log entry
      console.log("Activity:", data.activity);
      break;

    case "connection_status":
      // Connection status change
      console.log("Connection:", data.activity.type);
      break;
  }
};

// Send commands via WebSocket
ws.send(
  JSON.stringify({
    action: "send_command",
    data: {
      command: "AT",
      data: "ID",
    },
  })
);
```

### AI/LLM Endpoints

#### Chat with Local LLM

```http
POST /api/chat
Content-Type: application/json

{
  "message": "Show me the latest telemetry data"
}
```

## Data Formats

### Telemetry Data Structure

```json
{
  "id": 1,
  "TEAM_ID": "TEJAS",
  "mission_time_s": 123.45,
  "packet_count": 100,
  "altitude": 1234.56,
  "pressure": 1013.25,
  "temperature": 23.4,
  "voltage": 7.2,
  "gnss_time": "12:34:56",
  "latitude": 12.3456,
  "longitude": 78.9012,
  "gps_altitude": 1235.0,
  "satellites": 8,
  "accel_x": 0.1,
  "accel_y": 0.2,
  "accel_z": 9.8,
  "gyro_spin_rate": 0.5,
  "flight_state": 1,
  "gyro_x": 0.01,
  "gyro_y": 0.02,
  "gyro_z": 0.03,
  "roll": 1.2,
  "pitch": 0.8,
  "yaw": 45.6,
  "mag_x": 0.3,
  "mag_y": 0.4,
  "mag_z": 0.5,
  "humidity": 65.2,
  "current": 0.85,
  "power": 6.12,
  "baro_altitude": 1234.0,
  "air_quality_raw": 512,
  "aq_ethanol_ppm": 2.5,
  "mcu_temp_c": 25.6,
  "rssi_dbm": -45,
  "health_flags": "OK",
  "rtc_epoch": 1633024800,
  "cmd_echo": "AT+ID"
}
```

### Statistics Response

```json
{
  "packetRate": 1.5,
  "packetsReceived": 1500,
  "packetsSent": 25,
  "lastUpdate": "2023-10-01T12:00:00Z",
  "startTime": "2023-10-01T10:00:00Z",
  "lastCommand": "START",
  "lastCommandEcho": "OK",
  "frameStats": {
    "telemetryCount": 1400,
    "commandEchoCount": 25,
    "logEntryCount": 50,
    "unknownCount": 25
  }
}
```

## XBee Protocol Support

### Supported Frame Types

- **AT Command (0x08)**: Send AT commands to local XBee
- **AT Response (0x88)**: Receive AT command responses
- **TX Request (0x10)**: Send data to remote XBee
- **RX Packet (0x90)**: Receive data from remote XBee
- **Explicit RX (0x91)**: Receive data with cluster information

### Cluster ID Mapping

- `0x0001`: TELEMETRY data
- `0x0002`: LOG entries
- `0x0003`: CMD_RESPONSE (command echoes)

## Mission Management

### Automatic Mission Creation

When a `START` command is sent:

1. Current mission (if any) is marked as inactive
2. New mission database is created in `missions/` directory
3. All subsequent telemetry data is stored in the new mission database
4. Statistics are reset for the new mission

### Mission Database Structure

- Each mission gets its own SQLite database
- Databases are named: `missions/mission_<timestamp>.db`
- All telemetry tables use the same schema
- Original `telemetry.db` remains as a backup/reference

## Development

### Project Structure

```
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── ai/                     # LLM integration
│   ├── api/                    # Original API handlers
│   ├── config/                 # Configuration management
│   ├── models/                 # Data models
│   └── xbee/                   # XBee functionality
│       ├── types.go            # Type definitions
│       ├── serial.go           # Serial port management
│       ├── frame_processor.go  # XBee frame processing
│       ├── service.go          # Main XBee service
│       └── handlers.go         # HTTP API handlers
├── database/
│   └── telemetry.go           # Database operations
├── missions/                  # Mission databases
└── store/                     # Frontend store (TypeScript)
```

### Adding New Features

1. Define types in `internal/xbee/types.go`
2. Implement logic in appropriate service files
3. Add API endpoints in `handlers.go`
4. Update WebSocket handlers for live features

## Troubleshooting

### Common Issues

**Serial Port Access**

- Ensure XBee device is properly connected
- Check port permissions on Linux/Mac
- Verify baud rate matches XBee configuration

**Database Issues**

- Check file permissions in project directory
- Ensure `missions/` directory exists
- Verify SQLite compatibility

**WebSocket Connection**

- Check CORS configuration
- Verify WebSocket URL format
- Ensure firewall allows connections

### Logging

The application logs all important events:

- XBee connection status
- Frame processing errors
- Database operations
- Mission state changes

Check console output for detailed information.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
