package sqlutil

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	orderByPattern = regexp.MustCompile(`(?i)order by\s+.*?(?:limit\s+\d+)?$`)
	limitPattern   = regexp.MustCompile(`(?i)limit\s+(\d+)`)

	aggWords      = []string{"group by", "avg(", "sum(", "min(", "max(", "count("}
	rangeWords    = []string{"between", "since", "from ", "range", "history", "historical", "all", "average", "avg", "min", "max", "sum", "count", "std", "variance"}
	earliestWords = []string{"earliest", "first", "beginning", "begining", "at the beginning", "start", "starting", "initial", "initially", "oldest", "earlier"}
	latestWords   = []string{"latest", "recent", "current", "now", "most recent", "newest", "last"}
)

var altitudePattern = regexp.MustCompile(`(?i)(?P<metric>[a-zA-Z ]+?)\s*(?:reading|value)?\s+at\s+(?P<value>-?\d+(?:\.\d+)?)\s*(?:meters?|m)?\s*altitude`)

var metricColumnMap = map[string]string{
	"temperature":     "temperature",
	"pressure":        "pressure",
	"voltage":         "voltage",
	"humidity":        "humidity",
	"current":         "current",
	"power":           "power",
	"altitude":        "altitude",
	"barometric":      "baro_altitude",
	"baro":            "baro_altitude",
	"mcu temperature": "mcu_temp_c",
	"mcu temp":        "mcu_temp_c",
	"ethanol":         "aq_ethanol_ppm",
}

func TryRuleBasedSQL(question string) (string, bool) {
	summaryPattern := regexp.MustCompile(`(?i)(?:summary|overview|describe|show|list)\s+(?:of\s+)?(?:the\s+)?first\s+(\d+)\s+(?:rows|records|entries)`)
	if matches := summaryPattern.FindStringSubmatch(question); len(matches) == 2 {
		if limit, err := strconv.Atoi(matches[1]); err == nil && limit > 0 {
			sql := `SELECT * FROM telemetry ORDER BY rtc_epoch ASC LIMIT ` + strconv.Itoa(limit) + ` -- rule_based summary_limit=` + strconv.Itoa(limit)
			return sql, true
		}
	}

	firstRowsPattern := regexp.MustCompile(`(?i)first\s+(\d+)\s+(?:rows|records|entries)`)
	if matches := firstRowsPattern.FindStringSubmatch(question); len(matches) == 2 {
		if limit, err := strconv.Atoi(matches[1]); err == nil && limit > 0 {
			sql := `SELECT * FROM telemetry ORDER BY rtc_epoch ASC LIMIT ` + strconv.Itoa(limit) + ` -- rule_based summary_limit=` + strconv.Itoa(limit)
			return sql, true
		}
	}

	if m := altitudePattern.FindStringSubmatch(question); len(m) == 3 {
		metricRaw := strings.TrimSpace(m[1])
		targetVal := m[2]

		metricKey := strings.ToLower(metricRaw)
		metricCol, ok := metricColumnMap[metricKey]
		if !ok {
			parts := strings.Fields(metricKey)
			if len(parts) > 0 {
				metricCol, ok = metricColumnMap[parts[len(parts)-1]]
			}
		}
		if ok {
			sql := "SELECT altitude, mission_time_s, rtc_epoch, " + metricCol + " AS value FROM telemetry ORDER BY ABS(altitude - " + targetVal + ") ASC, rtc_epoch DESC LIMIT 1 -- rule_based metric=" + metricCol + " target_altitude=" + targetVal
			return sql, true
		}
	}
	return "", false
}

func QuestionNeedsEarliest(question string) bool {
	lower := strings.ToLower(question)
	for _, keyword := range earliestWords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func QuestionNeedsLatest(question, sql string) bool {
	if QuestionNeedsEarliest(question) {
		return false
	}

	lowerQuestion := strings.ToLower(question)
	for _, keyword := range latestWords {
		if strings.Contains(lowerQuestion, keyword) {
			return true
		}
	}

	for _, keyword := range rangeWords {
		if strings.Contains(lowerQuestion, keyword) {
			return false
		}
	}

	lowerSQL := strings.ToLower(sql)
	if strings.Contains(lowerSQL, "abs(") {
		return false
	}

	for _, keyword := range aggWords {
		if strings.Contains(lowerSQL, keyword) {
			return false
		}
	}

	return true
}

func ExtractRequestedLimit(question string) (int, bool) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:first|top|last|latest|earliest)\s+(\d+)`),
		regexp.MustCompile(`(?i)(?:\b)(\d+)\s+(?:rows|records|entries)`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(question); len(matches) == 2 {
			if value, err := strconv.Atoi(matches[1]); err == nil && value > 0 {
				return value, true
			}
		}
	}

	return 0, false
}

func ExtractLimitFromSQL(sql string) (int, bool) {
	if matches := limitPattern.FindStringSubmatch(sql); len(matches) == 2 {
		if value, err := strconv.Atoi(matches[1]); err == nil && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func EnforceOrderClause(sql string, asc bool, limit int) string {
	sanitized := strings.TrimRight(strings.TrimSpace(sql), ";")
	var desiredLimit int
	if limit > 0 {
		desiredLimit = limit
	} else {
		if existing, ok := ExtractLimitFromSQL(sanitized); ok {
			desiredLimit = existing
		} else {
			desiredLimit = 1
		}
	}

	clause := "ORDER BY rtc_epoch "
	if asc {
		clause += "ASC"
	} else {
		clause += "DESC"
	}
	clause += " LIMIT " + strconv.Itoa(desiredLimit)

	sanitized = orderByPattern.ReplaceAllString(sanitized, "")
	sanitized = strings.TrimSpace(sanitized)
	if sanitized == "" {
		return clause
	}
	if strings.HasSuffix(sanitized, ")") {
		return sanitized + " " + clause
	}
	return sanitized + " " + clause
}

func StripSQLMetadata(sql string) (string, map[string]string) {
	meta := make(map[string]string)
	if idx := strings.Index(sql, "--"); idx >= 0 {
		sqlPart := strings.TrimSpace(sql[:idx])
		comment := strings.TrimSpace(sql[idx+2:])
		if comment != "" {
			fields := strings.Fields(comment)
			for _, field := range fields {
				if parts := strings.SplitN(field, "=", 2); len(parts) == 2 {
					meta[parts[0]] = parts[1]
				}
			}
		}
		return sqlPart, meta
	}
	return sql, meta
}

func JoinSQLWithMetadata(sql string, meta map[string]string) string {
	if len(meta) == 0 {
		return strings.TrimSpace(sql)
	}
	pairs := make([]string, 0, len(meta))
	for k, v := range meta {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.TrimSpace(sql) + " -- " + strings.Join(pairs, " ")
}
