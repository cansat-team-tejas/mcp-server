# Go Fiber Port Plan

## Goals

- Replace the existing Python FastAPI services with a single Go application that exposes the same HTTP surface (`/chat`, `/ask`, `/telemetry`, `/query`).
- Reimplement command detection, AI-assisted SQL generation, telemetry formatting, and SQLite access in Go.
- Keep compatibility with the existing `telemetry.db` schema and `db_schema.sql` initializer.

## Package Layout

```
goapp/
  go.mod
  cmd/
    server/
      main.go          # Fiber bootstrap, route wiring, startup checks
  internal/
    api/
      handlers.go      # HTTP handlers for chat/ask/query/telemetry
      models.go        # Request/response shapes
    commands/
      catalog.go       # Static GS command catalog definition
      detect.go        # DetectCommandRequest + helpers
    ai/
      client.go        # HuggingFace chat completion calls (SQL + narratives)
      sql.go           # SQL generation workflow + rule-based fallbacks
    sqlutil/
      rules.go         # Rule-based SQL helpers, order enforcement
      meta.go          # Metadata parsing utilities
    telemetry/
      db.go            # SQLite connection pool + primitives
      format.go        # Prompt-oriented result summaries
    config/
      config.go        # Environment loading (API key, DB path, port)
```

## Key Decisions

- Use environment variable `HUGGING_FACE_TOKEN` for the API bearer token to avoid embedding secrets.
- Initialize SQLite using `db_schema.sql` if the `telemetry` table is missing.
- Share a single `*sql.DB` instance across handlers with connection pooling enabled.
- Mirror Python analytics logic (earliest/latest detection, metadata hints, command formatting) to keep behavior consistent.
- Provide graceful error handling with structured JSON responses for Fiber routes.

## Open Items

- Decide whether to stream AI responses; initial implementation will return full text synchronously, matching Python behavior.
- Add unit tests (out-of-scope for first port but noted for follow-up).
