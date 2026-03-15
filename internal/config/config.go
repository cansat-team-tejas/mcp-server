package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               int
	DBPath             string
	SchemaPath         string
	SimulationSeedPath string
	GeminiAPIKey       string
	GeminiModel        string
	APIKey             string
	AllowedOrigins     []string
	ReadOnlyMode       bool
	MaxTelemetryRows   int
	StaticDir          string
}

const (
	defaultPort               = 8000
	defaultSchemaPath         = "db_schema.sql"
	defaultSimulationSeedPath = "simulation_seed.sql"
	defaultGeminiModel        = "gemini-1.5-flash"
	defaultMaxTelemetryRows   = 1000
)

func readBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func loadLocalEnvFiles() {
	// Best-effort loading for local development; existing env vars are preserved.
	candidates := []string{
		".env",
		".env.server",
		"mcp-server/.env",
		"infra/.env.server",
		"../infra/.env.server",
	}
	for _, file := range candidates {
		_ = godotenv.Load(file)
	}
}

func normalizeDBPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed
	}

	// Local Windows dev commonly reuses container envs with /data/... paths.
	// Remap those to a writable workspace path.
	if runtime.GOOS == "windows" && strings.HasPrefix(trimmed, "/data/") {
		return filepath.Join("databases", filepath.Base(trimmed))
	}

	return trimmed
}

func Load() Config {
	loadLocalEnvFiles()

	port := defaultPort
	if val := os.Getenv("PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			port = parsed
		}
	}

	// DB_PATH is optional. If not provided, the app starts without an active database.
	dbPath := normalizeDBPath(os.Getenv("DB_PATH"))

	maxTelemetryRows := defaultMaxTelemetryRows
	if val := os.Getenv("MAX_TELEMETRY_ROWS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			maxTelemetryRows = parsed
		}
	}

	schemaPath := os.Getenv("DB_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = defaultSchemaPath
	}

	simulationSeedPath := os.Getenv("SIMULATION_SEED_PATH")
	if simulationSeedPath == "" {
		simulationSeedPath = defaultSimulationSeedPath
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = defaultGeminiModel
	}

	apiKey := os.Getenv("API_KEY")
	readOnlyMode := readBoolEnv("READ_ONLY_MODE", false)

	var allowedOrigins []string
	if val := os.Getenv("ALLOWED_ORIGINS"); val != "" {
		for _, o := range strings.Split(val, ",") {
			if o = strings.TrimSpace(o); o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	return Config{
		Port:               port,
		DBPath:             dbPath,
		SchemaPath:         schemaPath,
		SimulationSeedPath: simulationSeedPath,
		GeminiAPIKey:       geminiAPIKey,
		GeminiModel:        geminiModel,
		APIKey:             apiKey,
		AllowedOrigins:     allowedOrigins,
		ReadOnlyMode:       readOnlyMode,
		MaxTelemetryRows:   maxTelemetryRows,
		StaticDir:          staticDir,
	}
}

func (c Config) ListenAddress() string {
	return fmt.Sprintf(":%d", c.Port)
}
