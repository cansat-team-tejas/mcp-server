package telemetry

import (
	"database/sql"
	"goapp/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

func EnsureSchema(dbPath string) (*gorm.DB, error) {
	// Use pure Go SQLite driver by opening with sql.Open first
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create GORM DB instance using the existing sql.DB connection
	db, err := gorm.Open(sqlite.Dialector{
		Conn: sqlDB,
	}, &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, err
	}

	// Auto migrate the schema
	if err := db.AutoMigrate(&models.Telemetry{}); err != nil {
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
