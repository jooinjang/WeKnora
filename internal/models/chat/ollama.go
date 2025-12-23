package chat

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/types"
	ollamaapi "github.com/ollama/ollama/api"
)

// OllamaChat implements Ollama-based chat
type OllamaChat struct {
	modelName     string
	modelID       string
	ollamaService *ollama.OllamaService
}

// NewOllamaChat creates an Ollama chat instance
func NewOllamaChat(config *ChatConfig, ollamaService *ollama.OllamaService) (*OllamaChat, error) {
	return &OllamaChat{
		modelName:     config.ModelName,
		modelID:       config.ModelID,
		ollamaService: ollamaService,
	}, nil
}

// convertMessages converts message format to Ollama API format
func (c *OllamaChat) convertMessages(messages []Message) []ollamaapi.Message {
	ollamaMessages := make([]ollamaapi.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollamaapi.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return ollamaMessages
}

// buildChatRequest builds chat request parameters
func (c *OllamaChat) buildChatRequest(messages []Message, opts *ChatOptions, isStream bool) *ollamaapi.ChatRequest {
	streamFlag := isStream

	chatReq := &ollamaapi.ChatRequest{
		Model:    c.modelName,
		Messages: c.convertMessages(messages),
		Stream:   &streamFlag,
		Options:  make(map[string]interface{}),
	}

	if opts != nil {
		if opts.Temperature > 0 {
			chatReq.Options["temperature"] = opts.Temperature
		}
		if opts.TopP > 0 {
			chatReq.Options["top_p"] = opts.TopP
		}
		if opts.MaxTokens > 0 {
			chatReq.Options["num_predict"] = opts.MaxTokens
		}
		if opts.Thinking != nil {
			chatReq.Think = &ollamaapi.ThinkValue{
				Value: *opts.Thinking,
			}
		}
	}

	return chatReq
}

// Chat performs non-streaming chat
func (c *OllamaChat) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error) {
	if err := c.ensureModelAvailable(ctx); err != nil {
		return nil, err
	}

	chatReq := c.buildChatRequest(messages, opts, false)

	logger.GetLogger(ctx).Infof("Sending chat request to model %s", c.modelName)

	var responseContent string
	var promptTokens, completionTokens int

	err := c.ollamaService.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
		responseContent = resp.Message.Content

		if resp.EvalCount > 0 {
			promptTokens = resp.PromptEvalCount
			completionTokens = resp.EvalCount - promptTokens
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}

	return &types.ChatResponse{
		Content: responseContent,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

// ChatStream performs streaming chat
func (c *OllamaChat) ChatStream(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (<-chan types.StreamResponse, error) {
	if err := c.ensureModelAvailable(ctx); err != nil {
		return nil, err
	}

	chatReq := c.buildChatRequest(messages, opts, true)

	logger.GetLogger(ctx).Infof("Sending streaming chat request to model %s", c.modelName)

	streamChan := make(chan types.StreamResponse)

	go func() {
		defer close(streamChan)

		err := c.ollamaService.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
			if resp.Message.Content != "" {
				streamChan <- types.StreamResponse{
					ResponseType: types.ResponseTypeAnswer,
					Content:      resp.Message.Content,
					Done:         false,
				}
			}

			if resp.Done {
				streamChan <- types.StreamResponse{
					ResponseType: types.ResponseTypeAnswer,
					Done:         true,
				}
			}

			return nil
		})
		if err != nil {
			logger.GetLogger(ctx).Errorf("Streaming chat request failed: %v", err)
			streamChan <- types.StreamResponse{
				ResponseType: types.ResponseTypeAnswer,
				Done:         true,
			}
		}
	}()

	return streamChan, nil
}

// ensureModelAvailable ensures the model is available
func (c *OllamaChat) ensureModelAvailable(ctx context.Context) error {
	logger.GetLogger(ctx).Infof("Ensuring model %s is available", c.modelName)
	return c.ollamaService.EnsureModelAvailable(ctx, c.modelName)
}

// GetModelName returns the model name
func (c *OllamaChat) GetModelName() string {
	return c.modelName
}

// GetModelID returns the model ID
func (c *OllamaChat) GetModelID() string {
	return c.modelID
}
