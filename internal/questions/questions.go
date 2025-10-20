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
	// Always process command requests first, even without a DB
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

	// If no DB is connected, try to answer as a basic question without telemetry context
	if db == nil {
		return answerBasicQuestion(ctx, question, client)
	}

	// If DB has no telemetry rows, try basic question handling
	if has, err := telemetry.HasData(ctx, db); err == nil && !has {
		return answerBasicQuestion(ctx, question, client)
	}

	// DB exists with data: attempt SQL generation and telemetry-based answer
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

	// If no data exists, return a clear message immediately
	if len(rows) == 0 {
		return Answer{Content: "No telemetry data found in the database. Please push some data first or create a new database with telemetry records."}, nil
	}

	contextSections := make([]string, 0, 2)
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

// answerBasicQuestion handles general questions without telemetry data
func answerBasicQuestion(ctx context.Context, question string, client *ai.Client) (Answer, error) {
	systemPrompt := ai.GetSystemPrompt() + "\n\nNote: No telemetry database is currently active. Answer the user's question in a general conversational manner. If the question requires specific telemetry data, politely explain that a database must be created and populated first."

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
