package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"goapp/internal/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func OpenDatabase(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory %q: %w", dir, err)
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
	}
	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			return nil, fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}

	return db, nil
}

func CloseDatabase(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func EnsureSchema(ctx context.Context, db *gorm.DB, schemaPath string) error {
	exists, err := tableExists(ctx, db, "telemetry")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if schemaPath == "" {
		if err := db.WithContext(ctx).Exec(database.TableSchema).Error; err != nil {
			return fmt.Errorf("apply embedded schema: %w", err)
		}
		return nil
	}

	schemaBytes, err := readSchema(schemaPath)
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(string(schemaBytes)).Error; err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func SeedSimulationData(ctx context.Context, db *gorm.DB, seedPath string) error {
	if seedPath == "" {
		return nil
	}
	hasData, err := HasData(ctx, db)
	if err != nil {
		return err
	}
	if hasData {
		return nil
	}

	seedBytes, err := readSchema(seedPath)
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(string(seedBytes)).Error; err != nil {
		return fmt.Errorf("apply simulation seed: %w", err)
	}
	return nil
}

func readSchema(schemaPath string) ([]byte, error) {
	candidates := []string{schemaPath}

	if !filepath.IsAbs(schemaPath) {
		candidates = append(candidates, filepath.Join("..", schemaPath))
		candidates = append(candidates, filepath.Join("..", "server", schemaPath))
	}

	var lastErr error
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("read schema file: %w", lastErr)
	}
	return nil, fmt.Errorf("read schema file: schema not found")
}

func tableExists(ctx context.Context, db *gorm.DB, table string) (bool, error) {
	var name string
	err := db.WithContext(ctx).Raw("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Row().Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return name == table, nil
}

func GetTelemetry(ctx context.Context, db *gorm.DB) ([]map[string]any, error) {
	return Query(ctx, db, "SELECT * FROM telemetry ORDER BY packet_count ASC, id ASC LIMIT 1200")
}

func Query(ctx context.Context, db *gorm.DB, sql string) ([]map[string]any, error) {
	rows, err := db.WithContext(ctx).Raw(sql).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0)

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]any, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// HasData returns true if the telemetry table contains at least one row
func HasData(ctx context.Context, db *gorm.DB) (bool, error) {
	var exists int
	err := db.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM telemetry LIMIT 1)").Row().Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func TrimTelemetryHistory(ctx context.Context, db *gorm.DB, maxRows int) error {
	if maxRows <= 0 {
		return nil
	}
	return db.WithContext(ctx).Exec(
		"DELETE FROM telemetry WHERE id NOT IN (SELECT id FROM telemetry ORDER BY packet_count DESC, id DESC LIMIT ?)",
		maxRows,
	).Error
}

// GetCurrentSimulationRow returns the row corresponding to the current loop position
// in a preloaded simulation dataset, based on wall-clock time.
func GetCurrentSimulationRow(ctx context.Context, db *gorm.DB) (map[string]any, error) {
	if db == nil {
		return nil, nil
	}

	var maxPacket int
	if err := db.WithContext(ctx).Raw("SELECT COALESCE(MAX(packet_count), 0) FROM telemetry").Row().Scan(&maxPacket); err != nil {
		return nil, err
	}
	if maxPacket <= 0 {
		return nil, nil
	}

	currentPacket := int(math.Mod(float64(time.Now().UnixMilli()/100), float64(maxPacket))) + 1
	rows, err := Query(ctx, db, fmt.Sprintf("SELECT * FROM telemetry WHERE packet_count=%d ORDER BY id DESC LIMIT 1", currentPacket))
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return rows[0], nil
	}

	// Fallback to the latest row if packet-index lookup misses.
	rows, err = Query(ctx, db, "SELECT * FROM telemetry ORDER BY packet_count DESC, id DESC LIMIT 1")
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}
