// Package ai provides HTTP handlers for AI chat functionality with live streaming support.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// ChatHandlers provides HTTP handlers for AI chat endpoints
type ChatHandlers struct {
	client *Client
}

// NewChatHandlers creates new AI chat handlers
func NewChatHandlers(client *Client) *ChatHandlers {
	return &ChatHandlers{
		client: client,
	}
}

// APIChatRequest represents an incoming chat request from API
type APIChatRequest struct {
	Messages []Message `json:"messages" validate:"required,min=1"`
	Stream   bool      `json:"stream,omitempty"`
	Context  string    `json:"context,omitempty"` // telemetry, mission_planning, error_analysis, default
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// StreamMessage represents a streaming chat message
type StreamMessage struct {
	Type    string `json:"type"`            // "chunk", "done", "error"
	Content string `json:"content"`         // Message content chunk
	Done    bool   `json:"done"`            // Whether streaming is complete
	Error   string `json:"error,omitempty"` // Error message if any
}

// POST /api/chat - Handle chat requests (both streaming and non-streaming)
func (h *ChatHandlers) HandleChat(c *fiber.Ctx) error {
	var req APIChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(ChatResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	if len(req.Messages) == 0 {
		return c.Status(400).JSON(ChatResponse{
			Success: false,
			Error:   "At least one message is required",
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Determine context type from request or default to "default"
	contextType := "default"
	if req.Context != "" {
		contextType = req.Context
	}

	if req.Stream {
		// For streaming requests, upgrade to WebSocket is preferred
		// But we can also handle it with HTTP streaming
		return h.handleStreamingHTTP(c, ctx, req.Messages, contextType)
	}

	// Handle non-streaming request with context
	response, err := h.client.ChatWithContext(ctx, req.Messages, contextType)
	if err != nil {
		log.Printf("AI chat error: %v", err)
		return c.Status(500).JSON(ChatResponse{
			Success: false,
			Error:   "Failed to get AI response",
		})
	}

	return c.JSON(ChatResponse{
		Success: true,
		Message: response,
	})
}

// Handle HTTP streaming (Server-Sent Events)
func (h *ChatHandlers) handleStreamingHTTP(c *fiber.Ctx, ctx context.Context, messages []Message, contextType string) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	// Create a callback that writes to the response stream
	callback := func(chunk string) error {
		msg := StreamMessage{
			Type:    "chunk",
			Content: chunk,
			Done:    false,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		_, err = c.Write([]byte("data: " + string(data) + "\n\n"))
		if err != nil {
			return err
		}

		// Flush the response to ensure real-time streaming
		c.Context().Response.ConnectionClose()
		return nil
	}

	// Start streaming with context
	err := h.client.ChatStreamWithContext(ctx, messages, contextType, callback)

	// Send completion message
	var finalMsg StreamMessage
	if err != nil {
		finalMsg = StreamMessage{
			Type:  "error",
			Error: err.Error(),
			Done:  true,
		}
	} else {
		finalMsg = StreamMessage{
			Type: "done",
			Done: true,
		}
	}

	data, _ := json.Marshal(finalMsg)
	c.Write([]byte("data: " + string(data) + "\n\n"))

	return nil
}

// WebSocket handler for real-time chat streaming
func (h *ChatHandlers) HandleWebSocketChat(c *websocket.Conn) {
	defer c.Close()

	for {
		var req APIChatRequest
		err := c.ReadJSON(&req)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		if len(req.Messages) == 0 {
			c.WriteJSON(StreamMessage{
				Type:  "error",
				Error: "At least one message is required",
				Done:  true,
			})
			continue
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		// Determine context type from request or default to "default"
		contextType := "default"
		if req.Context != "" {
			contextType = req.Context
		}

		// Create callback that sends chunks via WebSocket
		callback := func(chunk string) error {
			return c.WriteJSON(StreamMessage{
				Type:    "chunk",
				Content: chunk,
				Done:    false,
			})
		}

		// Start streaming with context
		err = h.client.ChatStreamWithContext(ctx, req.Messages, contextType, callback)

		// Send completion message
		var finalMsg StreamMessage
		if err != nil {
			finalMsg = StreamMessage{
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			}
		} else {
			finalMsg = StreamMessage{
				Type: "done",
				Done: true,
			}
		}

		c.WriteJSON(finalMsg)
		cancel()
	}
}

// Middleware to check if request wants WebSocket upgrade for chat
func (h *ChatHandlers) WebSocketUpgrade(c *fiber.Ctx) error {
	// Check if this is a WebSocket upgrade request
	if websocket.IsWebSocketUpgrade(c) {
		return websocket.New(h.HandleWebSocketChat)(c)
	}

	// If not WebSocket, return error
	return c.Status(426).JSON(ChatResponse{
		Success: false,
		Error:   "WebSocket upgrade required for real-time chat",
	})
}

// RegisterRoutes registers all AI chat routes
func (h *ChatHandlers) RegisterRoutes(app *fiber.App) {
	ai := app.Group("/api")

	// Regular chat endpoint (supports both streaming and non-streaming)
	ai.Post("/chat", h.HandleChat)

	// WebSocket endpoint for real-time chat
	ai.Get("/chat/ws", h.WebSocketUpgrade)

	// Health check for AI service
	ai.Get("/chat/health", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{
			"success": true,
			"status":  "AI service is running",
			"model":   h.client.model,
		})
	})

	// Reload AI instructions endpoint
	ai.Post("/chat/reload-instructions", h.HandleReloadInstructions)
}

// HandleReloadInstructions reloads AI instructions from the configuration file
func (h *ChatHandlers) HandleReloadInstructions(c *fiber.Ctx) error {
	err := h.client.ReloadInstructions()
	if err != nil {
		return c.Status(500).JSON(ChatResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to reload instructions: %v", err),
		})
	}

	return c.JSON(ChatResponse{
		Success: true,
		Message: "AI instructions reloaded successfully",
	})
}
