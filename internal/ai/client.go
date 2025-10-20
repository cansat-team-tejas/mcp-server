package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// Ollama defaults
	defaultOllamaHost  = "http://localhost:11434"
	defaultOllamaModel = "gemma3:4b"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	// Ollama chat API expects: { model, messages: [{role, content}, ...], stream }
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Minimal struct to parse Ollama's non-streaming response
// https://github.com/ollama/ollama/blob/main/docs/api.md#generate-a-chat-completion
type chatResponse struct {
	Model         string  `json:"model"`
	CreatedAt     string  `json:"created_at"`
	Message       Message `json:"message"`
	Done          bool    `json:"done"`
	TotalDuration int64   `json:"total_duration"`
	EvalCount     int     `json:"eval_count"`
	Error         string  `json:"error"`
}

type Client struct {
	httpClient *http.Client
	host       string
	model      string
}

func NewClient(token string) *Client {
	// Allow overrides via env vars for flexibility
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = defaultOllamaHost
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = defaultOllamaModel
	}
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		host:       host,
		model:      model,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c == nil {
		return "", errors.New("ai client is nil")
	}

	payload := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	// POST {host}/api/chat
	url := c.host + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama error: %s", string(body))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if parsed.Error != "" {
		return "", fmt.Errorf("ollama error: %s", parsed.Error)
	}

	return parsed.Message.Content, nil
}
