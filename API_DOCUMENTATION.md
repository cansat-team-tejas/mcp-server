# API Documentation

## Base URL

Default: `http://localhost:8000`

## CORS Policy

**CORS restrictions have been removed.** All origins are allowed with the following methods:

- GET, POST, PUT, DELETE, OPTIONS

## Database Management

The system now supports **dynamic database switching**:

- All `.db` files are stored in the `databases/` directory
- When you create a new database, it automatically becomes the **active database**
- All subsequent operations (push data, queries, AI questions) use the **current active database**
- The main database is only used as a fallback on startup

## Endpoints

### 1. Push Telemetry Data

Insert a new telemetry data row into the database.

**Endpoint:** `POST /telemetry/push`

**Request Body:**

```json
{
  "TEAM_ID": "string",
  "mission_time_s": 0.0,
  "packet_count": 0,
  "altitude": 0.0,
  "pressure": 0.0,
  "temperature": 0.0,
  "voltage": 0.0,
  "gnss_time": "string",
  "latitude": 0.0,
  "longitude": 0.0,
  "gps_altitude": 0.0,
  "satellites": 0,
  "accel_x": 0.0,
  "accel_y": 0.0,
  "accel_z": 0.0,
  "gyro_spin_rate": 0.0,
  "flight_state": 0,
  "gyro_x": 0.0,
  "gyro_y": 0.0,
  "gyro_z": 0.0,
  "roll": 0.0,
  "pitch": 0.0,
  "yaw": 0.0,
  "mag_x": 0.0,
  "mag_y": 0.0,
  "mag_z": 0.0,
  "humidity": 0.0,
  "current": 0.0,
  "power": 0.0,
  "baro_altitude": 0.0,
  "air_quality_raw": 0,
  "aq_ethanol_ppm": 0.0,
  "mcu_temp_c": 0.0,
  "rssi_dbm": 0,
  "health_flags": "string",
  "rtc_epoch": 0,
  "cmd_echo": "string"
}
```

**Response (201 Created):**

```json
{
  "message": "Data inserted successfully",
  "id": 123
}
```

**Error Response (400/500):**

```json
{
  "error": "error description"
}
```

---

### 2. Create New Database

Create a new SQLite database with the telemetry schema. **This database automatically becomes the active database** for all subsequent operations.

### 1. Push Telemetry Data (JSON only)

Insert a new telemetry data row into the current active database using JSON.

**Request Body:**

```json
{
  "db_path": "my_mission.db"
}
```

**Note:** The database will be created in the `databases/` directory. You only need to provide the filename.

**Response (201 Created):**

```json
{
  "message": "Database created successfully and set as current",
  "db_path": "databases/my_mission.db",
  "full_path": "C:/full/path/to/databases/my_mission.db",
  "now_active": true
}
```

**Error Response (400/500):**

```json
{
  "error": "error description"
}
```

---

### 3. Get Current Database

Check which database is currently active.

**Endpoint:** `GET /database/current`

**Response (200 OK):**

```json
{
  "current_db_path": "databases/my_mission.db"
  "cmd_echo": "string",
  "log_data": "optional free-form log string"
```

---

### 4. Get All Telemetry Data

Retrieve all telemetry records from the **current active database**.

**Endpoint:** `GET /telemetry`

**Response (200 OK):**

```json
[
  {
    "id": 1,
    "TEAM_ID": "TEAM_001",
    "mission_time_s": 123.45,
    "packet_count": 100,
    "altitude": 1500.5,
    ...
  }
]
```

---

### 5. Ask Question (Natural Language Query)

Ask a natural language question about the telemetry data in the **current active database**. The AI will generate SQL and provide insights.

**Endpoint:** `POST /ask`

**Request Body:**

```json
{
  "question": "What was the maximum altitude reached?"
}
```

**Response (200 OK):**

```json
{
  "answer": {
    "content": "The maximum altitude reached was 2500.5 meters at mission time 345.2 seconds..."
  },
  "command": "optional_command_code"
}
```

---

### 6. Chat (Natural Language Query via GET)

Similar to /ask but uses query parameters. Uses the **current active database**.

**Endpoint:** `GET /chat?prompt=your+question+here`

**Response (200 OK):**

```json
{
  "answer": {
    "content": "AI response here..."
  }
}
```

---

### 7. Execute SQL Query

Execute a custom SELECT query on the **current active database**.

**Endpoint:** `POST /query`

**Request Body:**

```json
{
  "sql": "SELECT * FROM telemetry WHERE altitude > 1000 LIMIT 10"
}
```

**Response (200 OK):**

```json
{
  "result": [
    {
      "id": 1,
      "altitude": 1500.5,
      ...
    }
  ]
}
```

**Note:** Only SELECT queries are allowed for security reasons.

---

## Example Usage

### Workflow Example

```bash
# 1. First, create a new database
curl -X POST http://localhost:8000/database/create \
  -H "Content-Type: application/json" \
  -d '{"db_path": "mission_001.db"}'

# 2. Check current database
curl http://localhost:8000/database/current

# 3. Push data to the active database
curl -X POST http://localhost:8000/telemetry/push \
  -H "Content-Type: application/json" \
  -d '{ ... telemetry data ... }'

# 4. Query the active database
curl -X POST http://localhost:8000/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What is the maximum altitude?"}'
```

### Push Data Example (curl)

```bash
curl -X POST http://localhost:8000/telemetry/push \
  -H "Content-Type: application/json" \
  -d '{
    "TEAM_ID": "TEAM_001",
    "mission_time_s": 123.45,
    "packet_count": 100,
    "altitude": 1500.5,
    "pressure": 101325,
    "temperature": 25.5,
    "voltage": 3.7,
    "gnss_time": "2025-10-21T12:00:00Z",
    "latitude": 40.7128,
    "longitude": -74.0060,
    "gps_altitude": 1510.0,
    "satellites": 8,
    "accel_x": 0.1,
    "accel_y": 0.2,
    "accel_z": 9.8,
    "gyro_spin_rate": 0.5,
    "flight_state": 2,
    "gyro_x": 0.1,
    "gyro_y": 0.1,
    "gyro_z": 0.1,
    "roll": 5.0,
    "pitch": 10.0,
    "yaw": 180.0,
    "mag_x": 20.0,
    "mag_y": 30.0,
    "mag_z": 40.0,
    "humidity": 65.0,
    "current": 0.5,
    "power": 1.85,
    "baro_altitude": 1505.0,
    "air_quality_raw": 100,
    "aq_ethanol_ppm": 0.1,
    "mcu_temp_c": 35.0,
    "rssi_dbm": -70,
    "health_flags": "OK",
    "rtc_epoch": 1729512000,
    "cmd_echo": "CMD_OK"
  }'
```

### Create Database Example (curl)

```bash
# Create a new database (will be placed in databases/ directory)
curl -X POST http://localhost:8000/database/create \
  -H "Content-Type: application/json" \
  -d '{"db_path": "mission_001.db"}'

# Response:
# {
#   "message": "Database created successfully and set as current",
#   "db_path": "databases/mission_001.db",
#   "full_path": "C:/path/to/databases/mission_001.db",
#   "now_active": true
# }
```

### Get Current Database Example (curl)

```bash
curl http://localhost:8000/database/current

# Response:
# {
#   "current_db_path": "databases/mission_001.db"
# }
```

### Ask Question Example (curl)

```bash
curl -X POST http://localhost:8000/ask \
  -H "Content-Type: application/json" \
  -d '{
    "question": "What was the maximum altitude?"
  }'
```

## Database Schema

The database schema is defined in `internal/database/schema.go`. To modify the schema structure:

1. Edit the `TableSchema` constant in `internal/database/schema.go`
2. Update the `ColumnNames()` function if columns are added/removed
3. Update the `TelemetryRow` struct in `internal/api/models.go`
4. Update the INSERT query in `handlePushData` function

## AI Context

The AI context for question answering is defined in `internal/ai/context.go`. This file contains:

- System overview and mission context
- Database structure documentation
- Flight phase descriptions
- Data analysis guidelines
- Response style instructions

To modify the AI's behavior or knowledge, edit the constants in this file.
