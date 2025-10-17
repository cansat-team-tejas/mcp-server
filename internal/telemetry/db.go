package telemetry

import (
	"fmt"
	"os"
	"path/filepath"

	"goapp/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func EnsureSchema(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("database path is empty")
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(absPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	migrator := db.Migrator()
	if migrator.HasTable("telemetries") && !migrator.HasTable("telemetry") {
		if err := migrator.RenameTable("telemetries", "telemetry"); err != nil {
			return nil, err
		}
	}

	if migrator.HasTable("conversation_history") && !migrator.HasTable("conversation_histories") {
		if err := migrator.RenameTable("conversation_history", "conversation_histories"); err != nil {
			return nil, err
		}
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&models.Telemetry{}, &models.ConversationHistory{}); err != nil {
		return nil, err
	}

	return db, nil
}

func GetTelemetry(db *gorm.DB) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	if err := db.Find(&telemetries).Error; err != nil {
		return nil, err
	}
	return telemetries, nil
}

func Query(db *gorm.DB, sql string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	if err := db.Raw(sql).Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}
