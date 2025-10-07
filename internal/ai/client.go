// Package ai provides AI client functionality for natural language processing
// using Ollama local LLM with support for both regular and streaming responses.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	apiURL       = "http://localhost:11434/api/chat"
	defaultModel = "gemma3:4b"
)

// Message represents a chat message with role and content
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents the request payload for Ollama chat API
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ChatResponse represents a single response chunk from Ollama
type chatResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
}

// StreamCallback is called for each chunk of streaming response
type StreamCallback func(chunk string) error

// Client provides AI chat functionality using Ollama with instruction management
type Client struct {
	httpClient         *http.Client
	token              string
	model              string
	InstructionManager *InstructionManager
}

// NewClient creates a new AI client for Ollama communication with instruction management
func NewClient(token string) *Client {
	// Try to load AI instructions, but don't fail if not available
	instructionManager, err := NewInstructionManager("")
	if err != nil {
		fmt.Printf("Warning: Could not load AI instructions: %v\n", err)
		instructionManager = nil
	}

	return &Client{
		httpClient:         &http.Client{Timeout: 60 * time.Second},
		token:              token, // Token is optional for Ollama
		model:              defaultModel,
		InstructionManager: instructionManager,
	}
}

// Chat sends messages to Ollama and returns the complete response
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, buf)
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

// ChatStream sends messages to Ollama and streams the response in real-time
// The callback function is called for each chunk of the response
func (c *Client) ChatStream(ctx context.Context, messages []Message, callback StreamCallback) error {
	if c == nil {
		return errors.New("ai client is nil")
	}

	if callback == nil {
		return errors.New("callback function is required for streaming")
	}

	payload := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true, // Enable streaming
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	// Process streaming response line by line
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var parsed chatResponse
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			// Log the error but continue processing
			continue
		}

		if parsed.Error != "" {
			return fmt.Errorf("ollama error: %s", parsed.Error)
		}

		// Send the content chunk to the callback
		if parsed.Message.Content != "" {
			if err := callback(parsed.Message.Content); err != nil {
				return fmt.Errorf("callback error: %w", err)
			}
		}

		// Check if streaming is complete
		if parsed.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan response: %w", err)
	}

	return nil
}

// ChatWithContext sends messages to Ollama with a specific context type
func (c *Client) ChatWithContext(ctx context.Context, messages []Message, contextType string) (string, error) {
	if c.InstructionManager != nil {
		// Prepend system message based on context type
		systemMessage := Message{
			Role:    "system",
			Content: c.InstructionManager.BuildSystemMessage(contextType, true),
		}
		messages = append([]Message{systemMessage}, messages...)
	}

	return c.Chat(ctx, messages)
}

// ChatStreamWithContext streams AI responses with a specific context type
func (c *Client) ChatStreamWithContext(ctx context.Context, messages []Message, contextType string, callback StreamCallback) error {
	if c.InstructionManager != nil {
		// Prepend system message based on context type
		systemMessage := Message{
			Role:    "system",
			Content: c.InstructionManager.BuildSystemMessage(contextType, true),
		}
		messages = append([]Message{systemMessage}, messages...)
	}

	return c.ChatStream(ctx, messages, callback)
}

// GetPromptTemplate returns a formatted prompt template
func (c *Client) GetPromptTemplate(templateName string, params map[string]string) string {
	if c.InstructionManager != nil {
		return c.InstructionManager.GetPromptTemplate(templateName, params)
	}
	return ""
}

// ReloadInstructions reloads AI instructions from file
func (c *Client) ReloadInstructions() error {
	if c.InstructionManager != nil {
		return c.InstructionManager.ReloadInstructions()
	}
	return fmt.Errorf("instruction manager not available")
}
