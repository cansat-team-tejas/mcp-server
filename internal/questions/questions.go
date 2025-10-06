package questions

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"goapp/internal/ai"
	"goapp/internal/commands"
	"goapp/internal/sqlutil"
	"goapp/internal/telemetry"
)

type Answer struct {
	Content  string
	Commands []string
}

func AnswerQuestion(ctx context.Context, question string, db *sql.DB, client *ai.Client) (Answer, error) {
	commandEntries := commands.DetectCommandRequest(question)
	if len(commandEntries) > 0 {
		codes := make([]string, 0, len(commandEntries))
		responses := make([]string, 0, len(commandEntries))
		for _, entry := range commandEntries {
			codes = append(codes, entry.Code)
			responses = append(responses, commands.FormatCommandResponse(entry))
		}
		return Answer{
			Content:  strings.Join(responses, "\n"),
			Commands: codes,
		}, nil
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

	contextSections := make([]string, 0, 2)
	if len(rows) > 0 {
		contextSections = append(contextSections, telemetry.FormatResultsForPrompt(question, rows))
	} else {
		contextSections = append(contextSections, "No rows were returned for this query.")
	}

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

	systemInstruction := "You are an engaging telemetry data assistant. Use the provided context to craft a friendly, insight-rich reply, reference concrete numbers, and explain what they mean. Present the answer in short paragraphs or bullet lists."
	userMessage := fmt.Sprintf("User question: %s\nSQL executed: %s\nContext:\n%s\nRespond conversationally while staying grounded in the data.", question, sql, contextText)

	message := []ai.Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: userMessage},
	}

	response, err := client.Chat(ctx, message)
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
