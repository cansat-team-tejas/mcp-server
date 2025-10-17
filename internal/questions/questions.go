package questions

import (
	"context"
	"fmt"
	"goapp/internal/ai"
	"goapp/internal/commands"
	"goapp/internal/sqlutil"
	"strings"

	"gorm.io/gorm"
)

type Answer struct {
	Content  string
	Commands []string
}

func AnswerQuestion(ctx context.Context, question string, db *gorm.DB, client *ai.Client, fallbackContext string) (Answer, error) {
	// Handle commands first
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

	// Check if this is a general knowledge question using MCP-style classification
	needsData, err := classifyQuestionWithMCP(ctx, client, question)
	if err != nil {
		needsData = true // Default to requiring data if classification fails
	}

	if !needsData {
		// Use MCP-style context-aware AI response for general questions
		response, err := handleGeneralQuestionWithMCP(ctx, client, question)
		if err != nil {
			return Answer{}, fmt.Errorf("llm response: %w", err)
		}
		return Answer{Content: response}, nil
	}

	// If no database available, use fallback context
	if db == nil {
		response, err := handleFallbackResponse(ctx, client, question, fallbackContext)
		if err != nil {
			return Answer{}, fmt.Errorf("fallback response: %w", err)
		}
		return Answer{Content: response}, nil
	}

	// Handle data-requiring questions with MCP-style data context
	return handleDataQuestionWithMCP(ctx, client, db, question)
}

// classifyQuestionWithMCP uses MCP-style classification for question routing
func classifyQuestionWithMCP(ctx context.Context, client *ai.Client, question string) (bool, error) {
	systemPrompt := `You are an MCP (Model Context Protocol) classifier for a CanSat telemetry system.
Your role is to route questions to the appropriate context handler.

Classification Rules:
- DATA: Questions requiring telemetry database access, sensor readings, measurements
- GENERAL: Conceptual questions, definitions, greetings, explanations

Respond with exactly one word: DATA or GENERAL

Examples:
"What is the latest altitude?" → DATA
"Hi there!" → GENERAL
"Explain CanSat systems" → GENERAL
"Show temperature trends" → DATA`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Classify: %s", question)},
	}

	response, err := client.Chat(ctx, messages)
	if err != nil {
		return true, err
	}

	response = strings.TrimSpace(strings.ToUpper(response))
	return response == "DATA", nil
}

// handleGeneralQuestionWithMCP handles general knowledge questions using MCP context
func handleGeneralQuestionWithMCP(ctx context.Context, client *ai.Client, question string) (string, error) {
	// Use the AI instructions system with default context for general questions
	return client.ChatWithContext(ctx, []ai.Message{
		{Role: "user", Content: question},
	}, "default")
}

// handleFallbackResponse handles questions when no database is available
func handleFallbackResponse(ctx context.Context, client *ai.Client, question string, fallbackContext string) (string, error) {
	systemInstruction := "You are an engaging telemetry data assistant. Use the provided context to craft a friendly, insight-rich reply, reference concrete numbers, and explain what they mean. Present the answer in short paragraphs or bullet lists."
	userMessage := fmt.Sprintf("User question: %s\nContext:\n%s\nRespond conversationally while staying grounded in the data.", question, fallbackContext)

	messages := []ai.Message{
		{Role: "system", Content: systemInstruction},
		{Role: "user", Content: userMessage},
	}

	return client.Chat(ctx, messages)
}

// handleDataQuestionWithMCP handles data-requiring questions with MCP-style context
func handleDataQuestionWithMCP(ctx context.Context, client *ai.Client, db *gorm.DB, question string) (Answer, error) {
	// Generate SQL using the existing method
	rawSQL, err := client.GenerateSQL(ctx, question)
	if err != nil {
		return Answer{}, fmt.Errorf("generate sql: %w", err)
	}

	sql, _ := sqlutil.StripSQLMetadata(rawSQL)
	if sql == "" || !strings.HasPrefix(strings.ToLower(sql), "select") {
		return Answer{Content: fmt.Sprintf("AI could not generate a valid SQL query.\nAI output: %s", rawSQL)}, nil
	}

	// Query the database
	rows, err := queryDatabase(db, sql)
	if err != nil {
		return Answer{}, fmt.Errorf("run query: %w", err)
	}

	// Generate MCP-style response with telemetry context
	return generateMCPResponse(ctx, client, question, rows)
}

// queryDatabase executes the SQL query and returns results
func queryDatabase(db *gorm.DB, sql string) ([]map[string]interface{}, error) {
	rows, err := db.Raw(sql).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// generateMCPResponse creates a context-aware response using MCP principles
func generateMCPResponse(ctx context.Context, client *ai.Client, question string, data []map[string]interface{}) (Answer, error) {
	var contextStr string
	if len(data) == 0 {
		contextStr = "No data found for your query."
	} else {
		contextStr = formatDataForMCP(data)
	}

	// Use telemetry context for data-driven responses
	response, err := client.ChatWithContext(ctx, []ai.Message{
		{Role: "user", Content: fmt.Sprintf("Question: %s\n\nData Context:\n%s", question, contextStr)},
	}, "telemetry")

	if err != nil {
		return Answer{}, err
	}

	return Answer{Content: response}, nil
}

// formatDataForMCP formats query results for MCP context
func formatDataForMCP(data []map[string]interface{}) string {
	if len(data) == 0 {
		return "No data available."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d record(s):\n", len(data)))

	for i, row := range data {
		if i >= 5 { // Limit to first 5 records for context
			result.WriteString(fmt.Sprintf("... and %d more records\n", len(data)-i))
			break
		}

		result.WriteString(fmt.Sprintf("Record %d: ", i+1))
		var pairs []string
		for key, value := range row {
			if value != nil {
				pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
			}
		}
		result.WriteString(strings.Join(pairs, ", "))
		result.WriteString("\n")
	}

	return result.String()
}
