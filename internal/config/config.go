package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port             int
	DBPath           string
	SchemaPath       string
	HuggingFaceToken string
}

const (
	defaultPort       = 8000
	defaultDBPath     = "telemetry.db"
	defaultSchemaPath = "db_schema.sql"
)

func Load() Config {
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

	schemaPath := os.Getenv("DB_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = defaultSchemaPath
	}

	token := os.Getenv("HUGGING_FACE_TOKEN")
	if token == "" {
		token = "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"
	}

	return Config{
		Port:             port,
		DBPath:           dbPath,
		SchemaPath:       schemaPath,
		HuggingFaceToken: token,
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
