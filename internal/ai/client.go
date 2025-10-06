package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiURL       = "https://router.huggingface.co/v1/chat/completions"
	defaultModel = "meta-llama/Llama-3.1-8B-Instruct:fireworks-ai"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type Client struct {
	httpClient *http.Client
	token      string
	model      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		token:      token,
		model:      defaultModel,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c == nil {
		return "", errors.New("ai client is nil")
	}
	if c.token == "" {
		return "", errors.New("HUGGING_FACE_TOKEN is not set")
	}

	payload := ChatRequest{
		Messages: messages,
		Model:    c.model,
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

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
		return "", fmt.Errorf("huggingface error: %s", string(body))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if parsed.Error != nil {
		return "", fmt.Errorf("huggingface error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return "", errors.New("no choices returned from model")
	}

	return parsed.Choices[0].Message.Content, nil
}
