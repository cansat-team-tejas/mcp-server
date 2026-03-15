package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultGeminiEndpoint = "https://generativelanguage.googleapis.com/v1beta"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  map[string]interface{} `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type Client struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	model      string
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		endpoint:   defaultGeminiEndpoint,
		apiKey:     apiKey,
		model:      model,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c == nil {
		return "", errors.New("ai client is nil")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return "", errors.New("gemini api key is not configured")
	}

	var systemParts []string
	contents := make([]geminiContent, 0, len(messages))
	for _, message := range messages {
		if strings.TrimSpace(message.Content) == "" {
			continue
		}
		if message.Role == "system" {
			systemParts = append(systemParts, message.Content)
			continue
		}
		role := "user"
		if message.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: message.Content}},
		})
	}
	if len(contents) == 0 {
		return "", errors.New("no content provided to gemini")
	}

	payload := geminiRequest{
		Contents: contents,
		GenerationConfig: map[string]interface{}{
			"temperature":     0.2,
			"topP":            0.8,
			"candidateCount":  1,
			"maxOutputTokens": 1024,
		},
	}
	if len(systemParts) > 0 {
		payload.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: strings.Join(systemParts, "\n\n")}},
		}
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	requestURL := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.endpoint, url.PathEscape(c.model), url.QueryEscape(c.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, buf)
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
		return "", fmt.Errorf("gemini api error: %s", string(body))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if parsed.Error != nil {
		return "", fmt.Errorf("gemini api error: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini returned no candidates")
	}

	parts := make([]string, 0, len(parsed.Candidates[0].Content.Parts))
	for _, part := range parsed.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("gemini returned empty content")
	}

	return strings.Join(parts, "\n"), nil
}
