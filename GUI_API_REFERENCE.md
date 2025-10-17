# MCP Server API & WebSocket Reference

This guide outlines the HTTP endpoints and WebSocket message contracts exposed by the MCP server. Share this with the GUI/AI team so they can integrate safely with the ground software.

---

## Mission file conventions

- Every mission stores data in a dedicated SQLite database located under the `missions/` directory.
- File names should be provided **without** path separators. The server ensures each mission file ends with `.db`.
- Example mission filename: `mission_2025-10-10.db`.

---

## HTTP endpoints

All HTTP endpoints accept and return JSON. Errors return an object with an `error` string and an appropriate HTTP status code.

### `POST /ask`

Ask the AI assistant questions about the current mission data.

**Request body**

```json
{
  "question": "string, required",
  "filename": "mission database name, required"
}
```

**Successful response**

```json
{
  "answer": {
    "content": "Natural-language answer string"
  },
  "command": "Optional command suggestion"
}
```

- `command` is present when the AI suggests a single action (e.g., `DEPLOY`); otherwise it is omitted.

**Error responses**

- `400` when the payload is malformed or the question is blank.
- `500` when the AI or database layer fails. Body: `{ "error": "description" }`.

### `POST /data`

Fetch the most recent telemetry rows stored for a mission.

**Request body**

```json
{
  "filename": "mission database name, required"
}
```

**Successful response**

- Array of telemetry objects. Each telemetry record mirrors the columns in `internal/models/telemetry.go`. Example (fields omitted for brevity):

```json
[
  {
    "id": 123,
    "team_id": "TEAM42",
    "mission_time_s": 153.2,
    "packet_count": 87,
    "altitude": 1023.5,
    "gyro_spin_rate": 1.23,
    "yaw_rate_target": null,
    "created_at": "2025-10-10T12:34:56Z"
  }
]
```

**Error responses**

- `400` when `filename` is missing or invalid.
- `500` when the mission database cannot be opened or queried. Body: `{ "error": "description" }`.

---

## WebSocket API

Connect to `ws://<host>:<port>/api/xbee/ws` for live telemetry, command echoes, and mission activity. The server requires a standard WebSocket upgrade request; non-upgrade requests receive HTTP 426.

Messages use the following envelope:

```json
{
  "action": "string",
  "data": { "key": "value" }
}
```

Server responses share this structure:

```json
{
  "type": "string",
  "success": true,
  "data": {
    /* payload differs per type */
  }
}
```

In addition, the server streams telemetry, stats, and activity updates asynchronously using the `LiveTelemetryData` schema (see **Server push events**).

### Client actions

| Action          | Payload (`data`)                                        | Description                                                                            |
| --------------- | ------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `send_command`  | `{ "command": "text", "data": "optional raw payload" }` | Sends an AT command or data packet to the vehicle. `data` is interpreted as raw bytes. |
| `start_mission` | `{ "name": "optional mission name" }`                   | Starts a new mission. When omitted, the server auto-generates a timestamped name.      |
| `get_stats`     | `{}`                                                    | Requests an immediate stats snapshot.                                                  |
| `get_activity`  | `{ "limit": number (optional, default 50) }`            | Requests the latest activities up to `limit` entries.                                  |

### Command responses

```json
{
  "type": "command_response",
  "success": true,
  "error": null
}
```

- On failure, `success` is `false` and `error` contains the description.

### Mission responses

```json
{
  "type": "mission_response",
  "success": true,
  "data": {
    "id": "mission_1696966400",
    "name": "Mission_2025-10-10_12-00-00",
    "startTime": "2025-10-10T12:00:00Z",
    "isActive": true,
    "dbPath": "missions/mission_1696966400.db"
  }
}
```

### Stats responses

```json
{
  "type": "stats_response",
  "success": true,
  "data": {
    "packetRate": 2.5,
    "packetsReceived": 123,
    "packetsSent": 45,
    "lastUpdate": "2025-10-10T12:05:00Z",
    "frameStats": {
      "telemetryCount": 120,
      "logEntryCount": 3,
      "commandEchoCount": 0,
      "unknownCount": 0
    },
    "connectionStatus": "connected",
    "lastDataReceived": "2025-10-10T12:04:58Z"
  }
}
```

### Activity responses

```json
{
  "type": "activity_response",
  "success": true,
  "data": [
    {
      "timestamp": "2025-10-10T12:04:57Z",
      "type": "FRAME_RECEIVED",
      "frameType": "TELEMETRY",
      "details": "Received 64-byte telemetry frame"
    }
  ]
}
```

### Error envelope

Any unknown action or server-side failure returns:

```json
{
  "type": "error",
  "success": false,
  "error": "Description of the failure"
}
```

---

## Server push events

Independently of client requests, the server sends real-time updates with this schema:

```json
{
  "type": "<event type>",
  "timestamp": "ISO-8601",
  "data": { ... },
  "stats": { ... },
  "activity": { ... },
  "error": "optional"
}
```

Depending on `type`, only some fields are populated:

| Event type          | Contents                                                                       |
| ------------------- | ------------------------------------------------------------------------------ |
| `live_telemetry`    | `data` contains the latest `Telemetry` record mirroring the `/data` structure. |
| `stats_update`      | `stats` includes the same structure as `stats_response.data`.                  |
| `activity`          | `activity` contains a single activity entry.                                   |
| `connection_status` | `activity` includes a short message about connect/disconnect events.           |

A telemetry event example:

```json
{
  "type": "live_telemetry",
  "timestamp": "2025-10-10T12:04:58Z",
  "data": {
    "mission_time_s": 158.7,
    "altitude": 1034.2,
    "roll": 0.12,
    "pitch": -0.05,
    "yaw": 1.56,
    "health_flags": "0x01"
  }
}
```

---

## Error handling strategy

- HTTP layer always returns meaningful status codes (`4xx` for client input issues, `5xx` for internal failures).
- WebSocket messages include an `error` string alongside `success: false`.
- Activity stream also surfaces error events with `type: "ERROR"` when parsing or database operations fail.

---

## Integration checklist for the GUI team

1. Allow users to select or name a mission database (`*.db`).
2. Warm up the AI using `POST /ask` with the selected mission when presenting question-answer workflows.
3. Subscribe to the WebSocket stream immediately after mission selection to receive `live_telemetry` and `stats_update` events.
4. Use `send_command` to issue commands; monitor `command_response` and activity events for feedback.
5. Refresh historical data with `POST /data` when the GUI needs tabular views or exports.
6. Handle `connection_status` events to alert operators when the radio link drops.
