package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port        int
	DBPath      string
	SchemaPath  string
	LLMToken    string // Optional token for local LLM
	LLMModel    string // Local LLM model name
	LLMEndpoint string // Local LLM endpoint
	XBeeConfig  XBeeConfig
}

type XBeeConfig struct {
	DefaultBaudRate int    `json:"defaultBaudRate"`
	DefaultDataBits int    `json:"defaultDataBits"`
	DefaultStopBits int    `json:"defaultStopBits"`
	DefaultParity   string `json:"defaultParity"`
	MissionDir      string `json:"missionDir"`
}

const (
	defaultPort        = 8000
	defaultSchemaPath  = "db_schema.sql"
	defaultLLMEndpoint = "http://localhost:11434"
	defaultLLMModel    = "llama3.1:8b"
	defaultDBFileName  = "telemetry.db"

	// XBee defaults
	defaultXBeeBaudRate = 115200
	defaultXBeeDataBits = 8
	defaultXBeeStopBits = 1
	defaultXBeeParity   = "none"
	defaultMissionDir   = "missions"
)

func Load() Config {
	missionDir := os.Getenv("MISSION_DIR")
	if missionDir == "" {
		missionDir = defaultMissionDir
	}
	missionDir = filepath.Clean(missionDir)
	if absMission, err := filepath.Abs(missionDir); err == nil {
		missionDir = absMission
	}
	if err := os.MkdirAll(missionDir, 0o755); err != nil {
		missionDir = defaultMissionDir
		_ = os.MkdirAll(missionDir, 0o755)
	}

	defaultDBPath := filepath.Join(missionDir, defaultDBFileName)

	port := defaultPort
	if val := os.Getenv("PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			port = parsed
		}
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		if resolved, ok := resolvePath(defaultDBPath); ok {
			dbPath = resolved
		} else {
			dbPath = defaultDBPath
		}
	}

	if absPath, err := filepath.Abs(dbPath); err == nil {
		dbPath = absPath
	}

	missionPrefix := missionDir + string(os.PathSeparator)
	if !strings.HasPrefix(dbPath, missionPrefix) && dbPath != filepath.Join(missionDir, filepath.Base(dbPath)) {
		dbPath = defaultDBPath
	}

	schemaPath := os.Getenv("DB_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = defaultSchemaPath
	}

	// Local LLM configuration
	llmToken := os.Getenv("LLM_TOKEN") // Optional for Ollama
	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		llmModel = defaultLLMModel
	}
	llmEndpoint := os.Getenv("LLM_ENDPOINT")
	if llmEndpoint == "" {
		llmEndpoint = defaultLLMEndpoint
	}

	// XBee configuration
	xbeeBaudRate := defaultXBeeBaudRate
	if val := os.Getenv("XBEE_BAUD_RATE"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			xbeeBaudRate = parsed
		}
	}

	xbeeDataBits := defaultXBeeDataBits
	if val := os.Getenv("XBEE_DATA_BITS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			xbeeDataBits = parsed
		}
	}

	xbeeStopBits := defaultXBeeStopBits
	if val := os.Getenv("XBEE_STOP_BITS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			xbeeStopBits = parsed
		}
	}

	xbeeParity := os.Getenv("XBEE_PARITY")
	if xbeeParity == "" {
		xbeeParity = defaultXBeeParity
	}

	return Config{
		Port:        port,
		DBPath:      dbPath,
		SchemaPath:  schemaPath,
		LLMToken:    llmToken,
		LLMModel:    llmModel,
		LLMEndpoint: llmEndpoint,
		XBeeConfig: XBeeConfig{
			DefaultBaudRate: xbeeBaudRate,
			DefaultDataBits: xbeeDataBits,
			DefaultStopBits: xbeeStopBits,
			DefaultParity:   xbeeParity,
			MissionDir:      missionDir,
		},
	}
}

func (c Config) ListenAddress() string {
	return fmt.Sprintf(":%d", c.Port)
}

func resolvePath(relative string) (string, bool) {
	candidates := []string{relative}
	if !filepath.IsAbs(relative) {
		candidates = append(candidates,
			filepath.Join("..", relative),
			filepath.Join("..", "server", relative),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}
