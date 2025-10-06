# Go Fiber Telemetry Service

This Go application replaces the original Python FastAPI services for the CanSat telemetry assistant. It exposes the same endpoints (`/chat`, `/ask`, `/telemetry`, `/query`) using [Fiber](https://gofiber.io/) and mirrors the original behavior for command detection, AI-assisted SQL generation, and SQLite-backed telemetry queries.

## Prerequisites

- Go 1.22+
- SQLite 3 database file (`telemetry.db`) located at the repository root
- `db_schema.sql` available at the repository root so the app can auto-initialize the database if missing
- Hugging Face API token with access to `meta-llama/Llama-3.1-8B-Instruct:fireworks-ai`

## Configuration

Set the following environment variables before running the server:

| Variable             | Description                                 | Default         |
| -------------------- | ------------------------------------------- | --------------- |
| `HUGGING_FACE_TOKEN` | Bearer token for Hugging Face Inference API | **required**    |
| `PORT`               | HTTP port to listen on                      | `8000`          |
| `DB_PATH`            | Path to `telemetry.db`                      | `telemetry.db`  |
| `DB_SCHEMA_PATH`     | Path to `db_schema.sql`                     | `db_schema.sql` |

## Running

```powershell
cd goapp
go mod tidy
go run ./cmd/server
```

The server listens on `http://localhost:8000` by default.

## Endpoints

- `GET /chat?prompt=` – quick command/answer path (returns command codes when detected)
- `POST /ask` – `{ "question": "..." }`, returns a conversational reply grounded in telemetry data
- `GET /telemetry` – dumps all telemetry rows from SQLite
- `POST /query` – `{ "sql": "SELECT ..." }`, limited to `SELECT` statements

## Notes

- The AI assistant mirrors the Python behavior including rule-based SQL shortcuts, earliest/latest detection, and metadata-driven hints.
- Command detection uses the same catalog as the ground station, ensuring code parity with the original implementation.
- The schema check runs on startup and applies `db_schema.sql` automatically if the `telemetry` table is missing.
