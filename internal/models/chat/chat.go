package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/runtime"
	"github.com/Tencent/WeKnora/internal/types"
)

// Tool represents a function/tool definition
type Tool struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatOptions defines chat options
type ChatOptions struct {
	Temperature         float64 `json:"temperature"`           // Temperature parameter
	TopP                float64 `json:"top_p"`                 // Top P parameter
	Seed                int     `json:"seed"`                  // Random seed
	MaxTokens           int     `json:"max_tokens"`            // Maximum token count
	MaxCompletionTokens int     `json:"max_completion_tokens"` // Maximum completion token count
	FrequencyPenalty    float64 `json:"frequency_penalty"`     // Frequency penalty
	PresencePenalty     float64 `json:"presence_penalty"`      // Presence penalty
	Thinking            *bool   `json:"thinking"`              // Whether to enable thinking
	Tools               []Tool  `json:"tools,omitempty"`       // Available tools list
	ToolChoice          string  `json:"tool_choice,omitempty"` // "auto", "required", "none", or specific tool
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`                   // Role: system, user, assistant, tool
	Content    string     `json:"content"`                // Message content
	Name       string     `json:"name,omitempty"`         // Function/tool name (for tool role)
	ToolCallID string     `json:"tool_call_id,omitempty"` // Tool call ID (for tool role)
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Tool calls (for assistant role)
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Chat defines the chat interface
type Chat interface {
	// Chat performs non-streaming chat
	Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error)

	// ChatStream performs streaming chat
	ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan types.StreamResponse, error)

	// GetModelName gets the model name
	GetModelName() string

	// GetModelID gets the model ID
	GetModelID() string
}

type ChatConfig struct {
	Source    types.ModelSource
	BaseURL   string
	ModelName string
	APIKey    string
	ModelID   string
}

// NewChat creates a chat instance
func NewChat(config *ChatConfig) (Chat, error) {
	var chat Chat
	var err error
	switch strings.ToLower(string(config.Source)) {
	case string(types.ModelSourceLocal):
		runtime.GetContainer().Invoke(func(ollamaService *ollama.OllamaService) {
			chat, err = NewOllamaChat(config, ollamaService)
		})
		if err != nil {
			return nil, err
		}
		return chat, nil
	case string(types.ModelSourceRemote):
		return NewRemoteAPIChat(config)
	default:
		return nil, fmt.Errorf("unsupported chat model source: %s", config.Source)
	}
}
