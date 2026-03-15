package questions

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"goapp/internal/ai"
	"goapp/internal/commands"
	"goapp/internal/sqlutil"
	"goapp/internal/telemetry"

	"gorm.io/gorm"
)

type Answer struct {
	Content  string
	Commands []string
}



var nonInstantQueryHints = regexp.MustCompile(`(?i)average|avg|min|max|trend|history|over\s+time|between|since|from|at\s+\d`)
var currentSnapshotIntent = regexp.MustCompile(`(?i)latest|current|right\s*now|now\b|status|how\s+is|how\s+are|performing|live`)
var greetingIntent = regexp.MustCompile(`(?i)^\s*(hi|hello|hey+|yo|sup|hola|namaste|heyy+)\s*!*\s*$`)

func getRowValue(row map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func shouldForceCurrentSnapshot(question string, currentRow map[string]any) bool {
	if currentRow == nil {
		return false
	}
	if nonInstantQueryHints.MatchString(question) {
		return false
	}
	if greetingIntent.MatchString(question) {
		return true
	}
	trimmed := strings.TrimSpace(strings.ToLower(question))
	if trimmed == "" {
		return true
	}
	// Short check-ins usually imply "what's current right now" in this UI.
	if len(strings.Fields(trimmed)) <= 3 {
		if !strings.Contains(trimmed, "average") && !strings.Contains(trimmed, "history") {
			return true
		}
	}
	return false
}


func AnswerQuestion(ctx context.Context, question string, db *gorm.DB, client *ai.Client, currentRow map[string]any) (Answer, error) {
	commandEntries := commands.DetectCommandRequest(question)

	var commandContext string
	var commandCodes []string
	if len(commandEntries) > 0 {
		codes := make([]string, 0, len(commandEntries))
		labels := make([]string, 0, len(commandEntries))
		for _, entry := range commandEntries {
			codes = append(codes, entry.Code)
			labels = append(labels, fmt.Sprintf("%s (%s): %s", entry.Label, entry.Code, entry.Description))
		}
		commandCodes = codes
		commandContext = "Potentially relevant system commands detected:\n" + strings.Join(labels, "\n")
	}


	if db == nil {

		ans, err := answerBasicQuestion(ctx, question, client, commandContext)
		if err == nil {
			ans.Commands = commandCodes
		}
		return ans, err
	}

	if has, err := telemetry.HasData(ctx, db); err == nil && !has {

		ans, err := answerBasicQuestion(ctx, question, client, commandContext)
		if err == nil {
			ans.Commands = commandCodes
		}
		return ans, err
	}


	if shouldForceCurrentSnapshot(question, currentRow) || currentSnapshotIntent.MatchString(question) {

		ans, err := answerWithCurrentRowOnly(ctx, question, client, currentRow, commandContext)
		if err == nil {
			ans.Commands = commandCodes
		}
		return ans, err
	}



	rawSQL, err := client.GenerateSQL(ctx, question)

	if err != nil {
		return Answer{}, fmt.Errorf("generate sql: %w", err)
	}

	sql, meta := sqlutil.StripSQLMetadata(rawSQL)
	if sql == "" || !strings.HasPrefix(strings.ToLower(sql), "select") {
		return Answer{Content: fmt.Sprintf("AI could not generate a valid SQL query.\nAI output: %s", rawSQL)}, nil
	}

	rows, err := telemetry.Query(ctx, db, sql)
	if err != nil {
		return Answer{}, fmt.Errorf("run query: %w", err)
	}

	if len(rows) == 0 {

		return Answer{Content: "No telemetry data is available in the shared simulation dataset right now."}, nil
	}

	contextSections := make([]string, 0, 3)
	if currentRow != nil {
		if payload, err := formatCurrentRowForContext(currentRow); err == nil {
			contextSections = append(contextSections, payload)
		}
	}
	contextSections = append(contextSections, telemetry.FormatResultsForPrompt(question, rows))

	if len(meta) > 0 {
		metaPairs := make([]string, 0, len(meta))
		for k, v := range meta {
			metaPairs = append(metaPairs, fmt.Sprintf("%s=%s", k, v))
		}
		if len(rows) > 0 {
			if targetAlt, ok := meta["target_altitude"]; ok {
				if firstAlt, okFirst := toFloat(rows[0]["altitude"]); okFirst {
					if target, err := strconv.ParseFloat(targetAlt, 64); err == nil {
						delta := firstAlt - target
						metaPairs = append(metaPairs, fmt.Sprintf("altitude_delta=%s", telemetry.FormatValue(delta)))
					}
				}
			}
		}
		contextSections = append(contextSections, "Metadata hints: "+strings.Join(metaPairs, "; "))
	}

	contextText := strings.Join(contextSections, "\n")

	// Use the AI context from the context package
	systemInstruction := ai.GetSystemPrompt()
	if commandContext != "" {
		systemInstruction += "\n\n" + commandContext
	}

	userMessage := fmt.Sprintf("User question: %s\nSQL executed: %s\nContext:\n%s\nRespond conversationally while staying grounded in the data.", question, sql, contextText)

	message := []ai.Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: userMessage},
	}

	response, err := client.Chat(ctx, message)
	if err != nil {
		return Answer{}, fmt.Errorf("llm response: %w", err)
	}

	return Answer{Content: response, Commands: commandCodes}, nil
}

// answerWithCurrentRowOnly handles greetings or status requests using only the current telemetry snapshot
func answerWithCurrentRowOnly(ctx context.Context, question string, client *ai.Client, currentRow map[string]any, commandCtx string) (Answer, error) {
	contextText := "No current telemetry snapshot available."
	if currentRow != nil {
		if payload, err := formatCurrentRowForContext(currentRow); err == nil {
			contextText = payload
		}
	}

	systemPrompt := ai.GetSystemPrompt()
	if commandCtx != "" {
		systemPrompt += "\n\n" + commandCtx
	}

	userMessage := fmt.Sprintf("User question: %s\n\nContext (Current Telemetry Snapshot):\n%s\n\nPlease respond conversationally as the mission control assistant. Reference the values above if relevant to the status or just greet the user.", question, contextText)

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return Answer{}, fmt.Errorf("llm response: %w", err)
	}

	return Answer{Content: response}, nil
}

// answerBasicQuestion handles general questions without telemetry data
func answerBasicQuestion(ctx context.Context, question string, client *ai.Client, commandCtx string) (Answer, error) {
	systemPrompt := ai.GetSystemPrompt() + "\n\nNote: No telemetry database is currently active. Answer the user's question in a general conversational manner. If the question requires specific telemetry data, politely explain that a database must be created and populated first."
	if commandCtx != "" {
		systemPrompt += "\n\n" + commandCtx
	}

	userMessage := fmt.Sprintf("User question: %s\n\nPlease answer conversationally. If this question requires telemetry data to answer, explain that no database is currently connected.", question)

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return Answer{}, fmt.Errorf("llm response: %w", err)
	}

	return Answer{Content: response}, nil
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
		if v == "" {
			return 0, false
		}
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}

func formatCurrentRowForContext(row map[string]any) (string, error) {
	parts := []string{
		"Current simulation row snapshot (this is the row loaded right now):",
	}
	if v, ok := row["packet_count"]; ok {
		parts = append(parts, fmt.Sprintf("- packet_count: %s", telemetry.FormatValue(v)))
	}
	if v, ok := row["mission_time_s"]; ok {
		parts = append(parts, fmt.Sprintf("- mission_time_s: %s", telemetry.FormatValue(v)))
	}
	for _, field := range []string{"pressure", "temperature", "altitude", "voltage", "humidity", "current", "power", "rssi_dbm"} {
		if v, ok := row[field]; ok {
			parts = append(parts, fmt.Sprintf("- %s: %s", field, telemetry.FormatValue(v)))
		}
	}
	return strings.Join(parts, "\n"), nil
}
