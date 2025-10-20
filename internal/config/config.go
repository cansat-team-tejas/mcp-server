package config

import (
	"fmt"
	"os"
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
	defaultSchemaPath = "db_schema.sql"
)

func Load() Config {
	port := defaultPort
	if val := os.Getenv("PORT"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			port = parsed
		}
	}

	// DB_PATH is optional. If not provided, the app starts without an active database.
	dbPath := os.Getenv("DB_PATH")

	schemaPath := os.Getenv("DB_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = defaultSchemaPath
	}

	token := os.Getenv("HUGGING_FACE_TOKEN")

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

// no resolvePath needed when DB_PATH is optional
