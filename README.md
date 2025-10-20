# Go Fiber Telemetry Service

This Go application replaces the original Python FastAPI services for the CanSat telemetry assistant. It exposes the same endpoints (`/chat`, `/ask`, `/telemetry`, `/query`) using [Fiber](https://gofiber.io/) and mirrors the original behavior for command detection, AI-assisted SQL generation, and SQLite-backed telemetry queries.

## Prerequisites

- Go 1.22+
- SQLite 3 database file (`telemetry.db`) located at the repository root
- `db_schema.sql` available at the repository root so the app can auto-initialize the database if missing
- A local LLM via [Ollama](https://ollama.com/) running with the model `gemma3:4b`

Install and run Ollama (once):

```powershell
winget install Ollama.Ollama
ollama pull gemma3:4b
ollama serve
```

## Configuration

Environment variables (optional):

| Variable         | Description                        | Default                  |
| ---------------- | ---------------------------------- | ------------------------ |
| `PORT`           | HTTP port to listen on             | `8000`                   |
| `DB_PATH`        | Path to `telemetry.db`             | `telemetry.db`           |
| `DB_SCHEMA_PATH` | Path to `db_schema.sql`            | `db_schema.sql`          |
| `OLLAMA_HOST`    | Ollama host (protocol + host:port) | `http://localhost:11434` |
| `OLLAMA_MODEL`   | Model name to use in Ollama        | `gemma3:4b`              |

## Running

```powershell
go mod tidy
go run ./cmd
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
- LLM calls are directed to your local Ollama instance. You can switch models by setting `OLLAMA_MODEL`.
