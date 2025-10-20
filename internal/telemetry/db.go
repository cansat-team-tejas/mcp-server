package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"goapp/internal/database"
)

func EnsureSchema(ctx context.Context, db *sql.DB, schemaPath string) error {
	exists, err := tableExists(ctx, db, "telemetry")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// If schemaPath is empty, use the embedded schema from database package
	if schemaPath == "" {
		if _, err := db.ExecContext(ctx, database.TableSchema); err != nil {
			return fmt.Errorf("apply embedded schema: %w", err)
		}
		return nil
	}

	// Otherwise, read from file
	schemaBytes, err := readSchema(schemaPath)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, string(schemaBytes)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
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

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return name == table, nil
}

func GetTelemetry(ctx context.Context, db *sql.DB) ([]map[string]any, error) {
	return Query(ctx, db, "SELECT * FROM telemetry")
}

func Query(ctx context.Context, db *sql.DB, sql string) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, sql)
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
func HasData(ctx context.Context, db *sql.DB) (bool, error) {
	var exists int
	// Using EXISTS is efficient for presence checks
	err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM telemetry LIMIT 1)").Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}
