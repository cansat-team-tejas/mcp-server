package database

import (
	"log"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"

	"goapp/internal/models"
)

type Database struct {
	DB *gorm.DB
}

type TelemetryRepository struct {
	db *gorm.DB
}

type ConversationRepository struct {
	db *gorm.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	// Create absolute path for database
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent, // Change to logger.Info for debugging
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open database connection with pure Go SQLite driver
	db, err := gorm.Open(sqlite.Open(absPath+"?_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.Telemetry{}, &models.ConversationHistory{})
	if err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

func (d *Database) GetTelemetryRepository() *TelemetryRepository {
	return &TelemetryRepository{db: d.DB}
}

func (d *Database) GetConversationRepository() *ConversationRepository {
	return &ConversationRepository{db: d.DB}
}

// TelemetryRepository methods
func (r *TelemetryRepository) Create(telemetry *models.Telemetry) error {
	return r.db.Create(telemetry).Error
}

func (r *TelemetryRepository) GetAll() ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := r.db.Order("rtc_epoch desc").Find(&telemetries).Error
	return telemetries, err
}

func (r *TelemetryRepository) GetByTimeRange(startTime, endTime int64) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := r.db.Where("rtc_epoch BETWEEN ? AND ?", startTime, endTime).
		Order("rtc_epoch asc").
		Find(&telemetries).Error
	return telemetries, err
}

func (r *TelemetryRepository) GetLatest(limit int) ([]models.Telemetry, error) {
	var telemetries []models.Telemetry
	err := r.db.Order("rtc_epoch desc").Limit(limit).Find(&telemetries).Error
	return telemetries, err
}

func (r *TelemetryRepository) GetByPacketCount(count int) (*models.Telemetry, error) {
	var telemetry models.Telemetry
	err := r.db.Where("packet_count = ?", count).First(&telemetry).Error
	return &telemetry, err
}

func (r *TelemetryRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Telemetry{}).Count(&count).Error
	return count, err
}

func (r *TelemetryRepository) DeleteOlderThan(rtcEpoch int64) error {
	return r.db.Where("rtc_epoch < ?", rtcEpoch).Delete(&models.Telemetry{}).Error
}

// Statistics methods
func (r *TelemetryRepository) GetStatsByTimeRange(startTime, endTime int64) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get count
	var count int64
	if err := r.db.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ?", startTime, endTime).
		Count(&count).Error; err != nil {
		return nil, err
	}
	stats["total_packets"] = count

	// Get altitude range
	var maxAlt, minAlt, avgAlt float64
	if err := r.db.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ? AND altitude IS NOT NULL", startTime, endTime).
		Select("MAX(altitude) as max_alt, MIN(altitude) as min_alt, AVG(altitude) as avg_alt").
		Row().Scan(&maxAlt, &minAlt, &avgAlt); err != nil {
		return nil, err
	}
	stats["max_altitude"] = maxAlt
	stats["min_altitude"] = minAlt
	stats["avg_altitude"] = avgAlt

	// Get temperature range
	var maxTemp, minTemp, avgTemp float64
	if err := r.db.Model(&models.Telemetry{}).
		Where("rtc_epoch BETWEEN ? AND ? AND temperature IS NOT NULL", startTime, endTime).
		Select("MAX(temperature) as max_temp, MIN(temperature) as min_temp, AVG(temperature) as avg_temp").
		Row().Scan(&maxTemp, &minTemp, &avgTemp); err != nil {
		return nil, err
	}
	stats["max_temperature"] = maxTemp
	stats["min_temperature"] = minTemp
	stats["avg_temperature"] = avgTemp

	return stats, nil
}

// ConversationRepository methods
func (r *ConversationRepository) Create(conversation *models.ConversationHistory) error {
	return r.db.Create(conversation).Error
}

func (r *ConversationRepository) GetByMissionID(missionID string) ([]models.ConversationHistory, error) {
	var conversations []models.ConversationHistory
	err := r.db.Where("mission_id = ?", missionID).
		Order("timestamp ASC").
		Find(&conversations).Error
	return conversations, err
}

func (r *ConversationRepository) GetByMissionIDPaginated(missionID string, limit, offset int) ([]models.ConversationHistory, error) {
	var conversations []models.ConversationHistory
	err := r.db.Where("mission_id = ?", missionID).
		Order("timestamp ASC").
		Limit(limit).
		Offset(offset).
		Find(&conversations).Error
	return conversations, err
}

func (r *ConversationRepository) CountByMissionID(missionID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ConversationHistory{}).
		Where("mission_id = ?", missionID).
		Count(&count).Error
	return count, err
}

func (r *ConversationRepository) DeleteByMissionID(missionID string) error {
	return r.db.Where("mission_id = ?", missionID).Delete(&models.ConversationHistory{}).Error
}
