package telemetry

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

func FormatValue(value any) string {
	switch v := value.(type) {
	case float32:
		return fmt.Sprintf("%.2f", v)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Sprintf("%v", v)
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return fmt.Sprintf("%.2f", f)
		}
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func FormatResultsForPrompt(question string, rows []map[string]any) string {
	if len(rows) == 0 {
		return "No rows returned."
	}

	lines := []string{fmt.Sprintf("Rows returned: %d", len(rows))}

	missionTimes := extractFloatColumn(rows, "mission_time_s")
	if len(missionTimes) > 0 {
		sort.Float64s(missionTimes)
		lines = append(lines, fmt.Sprintf("Mission time range: %.2fs – %.2fs", missionTimes[0], missionTimes[len(missionTimes)-1]))
	}

	for _, field := range []string{"temperature", "altitude", "pressure", "voltage", "humidity", "current", "power"} {
		values := extractFloatColumn(rows, field)
		if len(values) == 0 {
			continue
		}
		avg := mean(values)
		min := values[0]
		max := values[0]
		for _, val := range values[1:] {
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
		lines = append(lines, fmt.Sprintf("%s: avg %.2f, min %.2f, max %.2f", field, avg, min, max))
	}

	previewCount := len(rows)
	if previewCount > 10 {
		previewCount = 10
	}

	highlightLines := make([]string, 0, previewCount)
	for i := 0; i < previewCount; i++ {
		row := rows[i]
		parts := make([]string, 0, 3)
		if val, ok := toFloat(row["mission_time_s"]); ok {
			parts = append(parts, fmt.Sprintf("t=%.2fs", val))
		} else if row["mission_time_s"] != nil {
			parts = append(parts, fmt.Sprintf("t=%v", row["mission_time_s"]))
		}
		if val, ok := toFloat(row["altitude"]); ok {
			parts = append(parts, fmt.Sprintf("alt=%.2fm", val))
		} else if row["altitude"] != nil {
			parts = append(parts, fmt.Sprintf("alt=%v", row["altitude"]))
		}
		if val, ok := toFloat(row["temperature"]); ok {
			parts = append(parts, fmt.Sprintf("temp=%.2f°C", val))
		} else if row["temperature"] != nil {
			parts = append(parts, fmt.Sprintf("temp=%v", row["temperature"]))
		}
		if len(parts) == 0 {
			parts = append(parts, "key metrics unavailable")
		}
		id := row["id"]
		highlightLines = append(highlightLines, fmt.Sprintf("Row %d (id=%v): %s", i+1, id, strings.Join(parts, ", ")))
	}

	if len(highlightLines) > 0 {
		lines = append(lines, "Highlights from the first rows:")
		for _, hl := range highlightLines {
			lines = append(lines, "- "+hl)
		}
	}

	previewRows := rows[:previewCount]
	if payload, err := json.MarshalIndent(previewRows, "", "  "); err == nil {
		lines = append(lines, "Raw data preview (first 10 rows max):")
		lines = append(lines, string(payload))
	}

	return strings.Join(lines, "\n")
}

func extractFloatColumn(rows []map[string]any, column string) []float64 {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		if val, ok := toFloat(row[column]); ok {
			values = append(values, val)
		}
	}
	return values
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case nil:
		return 0, false
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
		return 0, false
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}
