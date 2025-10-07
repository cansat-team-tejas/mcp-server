package ai

import (
	"context"
	"fmt"
	"strings"

	"goapp/internal/sqlutil"
)

func (c *Client) GenerateSQL(ctx context.Context, question string) (string, error) {
	if sql, ok := sqlutil.TryRuleBasedSQL(question); ok {
		return sql, nil
	}

	systemPrompt := "You are an assistant that writes SQLite SELECT statements for the telemetry table. " +
		"Always query from the table named `telemetry` and only use the columns listed in the schema. " +
		"If the user does not specify a timeframe, assume they want the most recent record and append `ORDER BY rtc_epoch DESC LIMIT 1`. " +
		"If they request the earliest or beginning of the data, use `ORDER BY rtc_epoch ASC LIMIT 1`. " +
		"When the question asks for a measurement at a specific altitude or other numeric value without exact matching rows, use `ORDER BY ABS(column - value)` to find the closest entry, falling back to rtc_epoch as a tiebreaker. " +
		"Return only the SQL text with no explanation."

	schema := "TEAM_ID, mission_time_s, packet_count, altitude, pressure, temperature, voltage, " +
		"gnss_time, latitude, longitude, gps_altitude, satellites, " +
		"accel_x, accel_y, accel_z, gyro_spin_rate, flight_state, " +
		"gyro_x, gyro_y, gyro_z, roll, pitch, yaw, " +
		"mag_x, mag_y, mag_z, humidity, " +
		"current, power, baro_altitude, " +
		"air_quality_raw, aq_ethanol_ppm, mcu_temp_c, rssi_dbm, health_flags, rtc_epoch, cmd_echo"

	userPrompt := "Schema columns: " + schema + "\nQuestion: " + question + "\nRespond with a single SQLite SELECT statement."

	baseMessages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	rawSQL, err := c.requestSQL(ctx, baseMessages)
	if err != nil {
		return "", err
	}

	sqlText, meta := sqlutil.StripSQLMetadata(rawSQL)

	if !isValidSQL(sqlText) {
		retryMessages := append(baseMessages, Message{Role: "system", Content: "Your previous output was invalid. Reply with only a corrected SQLite SELECT statement that queries from telemetry."})
		rawSQL, err = c.requestSQL(ctx, retryMessages)
		if err != nil {
			return "", err
		}
		sqlText, meta = sqlutil.StripSQLMetadata(rawSQL)
	}

	requestedLimit, hasRequestedLimit := sqlutil.ExtractRequestedLimit(question)

	if sqlutil.QuestionNeedsEarliest(question) {
		limit := 0
		if hasRequestedLimit {
			limit = requestedLimit
		}
		sqlText = sqlutil.EnforceOrderClause(sqlText, true, limit)
	} else if sqlutil.QuestionNeedsLatest(question, sqlText) {
		limit := 0
		if hasRequestedLimit {
			limit = requestedLimit
		}
		sqlText = sqlutil.EnforceOrderClause(sqlText, false, limit)
	}

	return sqlutil.JoinSQLWithMetadata(sqlText, meta), nil
}

func (c *Client) requestSQL(ctx context.Context, messages []Message) (string, error) {
	content, err := c.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	// Debug: Log the raw AI response
	fmt.Printf("DEBUG: Raw AI response: %q\n", content)

	cleaned := strings.TrimSpace(content)
	if idx := strings.Index(cleaned, "```"); idx != -1 {
		end := strings.LastIndex(cleaned, "```")
		if end > idx {
			cleaned = cleaned[idx+3 : end]
		} else {
			cleaned = strings.ReplaceAll(cleaned, "```", "")
		}
	}
	cleaned = strings.TrimSpace(cleaned)
	if strings.HasPrefix(strings.ToLower(cleaned), "sql") {
		cleaned = strings.TrimSpace(cleaned[3:])
	}

	// Debug: Log the cleaned response
	fmt.Printf("DEBUG: Cleaned SQL: %q\n", cleaned)

	return cleaned, nil
}

func isValidSQL(sql string) bool {
	lowered := strings.ToLower(strings.TrimSpace(sql))
	return strings.HasPrefix(lowered, "select") && strings.Contains(lowered, " from ")
}
