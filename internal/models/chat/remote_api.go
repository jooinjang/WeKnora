package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/sashabaranov/go-openai"
)

// RemoteAPIChat implements remote API-based chat
type RemoteAPIChat struct {
	modelName string
	client    *openai.Client
	modelID   string
	baseURL   string
	apiKey    string
}

// QwenChatCompletionRequest is the custom request structure for qwen models
type QwenChatCompletionRequest struct {
	openai.ChatCompletionRequest
	EnableThinking *bool `json:"enable_thinking,omitempty"` // qwen model specific field
}

// NewRemoteAPIChat creates a remote API chat instance
func NewRemoteAPIChat(chatConfig *ChatConfig) (*RemoteAPIChat, error) {
	apiKey := chatConfig.APIKey
	config := openai.DefaultConfig(apiKey)
	if baseURL := chatConfig.BaseURL; baseURL != "" {
		config.BaseURL = baseURL
	}
	return &RemoteAPIChat{
		modelName: chatConfig.ModelName,
		client:    openai.NewClientWithConfig(config),
		modelID:   chatConfig.ModelID,
		baseURL:   chatConfig.BaseURL,
		apiKey:    apiKey,
	}, nil
}

// convertMessages converts message format to OpenAI format
func (c *RemoteAPIChat) convertMessages(messages []Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role: msg.Role,
		}

		// Handle content: for assistant role, content may be empty (when there are tool_calls)
		if msg.Content != "" {
			openaiMsg.Content = msg.Content
		}

		// Handle tool calls (assistant role)
		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolType := openai.ToolType(tc.Type)
				openaiMsg.ToolCalls = append(openaiMsg.ToolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: toolType,
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}

		// Handle tool role messages (tool return results)
		if msg.Role == "tool" {
			openaiMsg.ToolCallID = msg.ToolCallID
			openaiMsg.Name = msg.Name
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}
	return openaiMessages
}

// isAliyunQwen3Model checks if it is a qwen model
func (c *RemoteAPIChat) isAliyunQwen3Model() bool {
	return strings.HasPrefix(c.modelName, "qwen3-") && c.baseURL == "https://dashscope.aliyuncs.com/compatible-mode/v1"
}

// isDeepSeekModel checks if it is a DeepSeek model
func (c *RemoteAPIChat) isDeepSeekModel() bool {
	return strings.Contains(strings.ToLower(c.modelName), "deepseek")
}

// buildQwenChatCompletionRequest builds chat request parameters for qwen models
func (c *RemoteAPIChat) buildQwenChatCompletionRequest(messages []Message,
	opts *ChatOptions, isStream bool,
) QwenChatCompletionRequest {
	req := QwenChatCompletionRequest{
		ChatCompletionRequest: c.buildChatCompletionRequest(messages, opts, isStream),
	}

	// For qwen models, force enable_thinking: false in non-streaming calls
	if !isStream {
		enableThinking := false
		req.EnableThinking = &enableThinking
	}
	return req
}

// buildChatCompletionRequest builds chat request parameters
func (c *RemoteAPIChat) buildChatCompletionRequest(messages []Message,
	opts *ChatOptions, isStream bool,
) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:    c.modelName,
		Messages: c.convertMessages(messages),
		Stream:   isStream,
	}
	thinking := false

	// Add optional parameters
	if opts != nil {
		if opts.Temperature > 0 {
			req.Temperature = float32(opts.Temperature)
		}
		if opts.TopP > 0 {
			req.TopP = float32(opts.TopP)
		}
		if opts.MaxTokens > 0 {
			req.MaxTokens = opts.MaxTokens
		}
		if opts.MaxCompletionTokens > 0 {
			req.MaxCompletionTokens = opts.MaxCompletionTokens
		}
		if opts.FrequencyPenalty > 0 {
			req.FrequencyPenalty = float32(opts.FrequencyPenalty)
		}
		if opts.PresencePenalty > 0 {
			req.PresencePenalty = float32(opts.PresencePenalty)
		}
		if opts.Thinking != nil {
			thinking = *opts.Thinking
		}

		// Handle Tools (function definitions)
		if len(opts.Tools) > 0 {
			req.Tools = make([]openai.Tool, 0, len(opts.Tools))
			for _, tool := range opts.Tools {
				toolType := openai.ToolType(tool.Type)
				openaiTool := openai.Tool{
					Type: toolType,
					Function: &openai.FunctionDefinition{
						Name:        tool.Function.Name,
						Description: tool.Function.Description,
					},
				}
				// Convert Parameters (map[string]interface{} -> JSON Schema)
				if tool.Function.Parameters != nil {
					// Parameters is already a JSON Schema format map, use directly
					openaiTool.Function.Parameters = tool.Function.Parameters
				}
				req.Tools = append(req.Tools, openaiTool)
			}
		}

		// Handle ToolChoice
		// ToolChoice can be a string or ToolChoice object
		// For "auto", "none", "required", use the string directly
		// For specific tool names, use ToolChoice object
		// Note: Some models (like DeepSeek) don't support tool_choice, need to skip setting
		if opts.ToolChoice != "" {
			// DeepSeek models don't support tool_choice, skip setting (default behavior will automatically use tools)
			if c.isDeepSeekModel() {
				// For DeepSeek, don't set tool_choice, let the API use default behavior
				// If there are tools, DeepSeek will automatically use them
				logger.Infof(context.Background(), "deepseek model, skip tool_choice")
			} else {
				switch opts.ToolChoice {
				case "none", "required", "auto":
					// Use string directly
					req.ToolChoice = opts.ToolChoice
				default:
					// Specific tool name, use ToolChoice object
					req.ToolChoice = openai.ToolChoice{
						Type: "function",
						Function: openai.ToolFunction{
							Name: opts.ToolChoice,
						},
					}
				}
			}
		}
	}

	req.ChatTemplateKwargs = map[string]interface{}{
		"enable_thinking": thinking,
	}

	// print req
	// jsonData, err := json.Marshal(req)
	// if err != nil {
	// 	logger.Error(context.Background(), "marshal request: %w", err)
	// }
	// logger.Infof(context.Background(), "llm request: %s", string(jsonData))

	return req
}

// Chat performs non-streaming chat
func (c *RemoteAPIChat) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error) {
	// If it's a qwen model, use custom request
	if c.isAliyunQwen3Model() {
		return c.chatWithQwen(ctx, messages, opts)
	}

	// Build request parameters
	req := c.buildChatCompletionRequest(messages, opts, false)

	// Send request
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	choice := resp.Choices[0]
	response := &types.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	// Convert Tool Calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]types.LLMToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			response.ToolCalls = append(response.ToolCalls, types.LLMToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: types.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return response, nil
}

// chatWithQwen handles qwen models with custom requests
func (c *RemoteAPIChat) chatWithQwen(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (*types.ChatResponse, error) {
	// Build qwen request parameters
	req := c.buildQwenChatCompletionRequest(messages, opts, false)

	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build URL
	endpoint := c.baseURL + "/chat/completions"

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set request headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var chatResp openai.ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	choice := chatResp.Choices[0]
	response := &types.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}

	// Convert Tool Calls
	if len(choice.Message.ToolCalls) > 0 {
		response.ToolCalls = make([]types.LLMToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			response.ToolCalls = append(response.ToolCalls, types.LLMToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: types.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return response, nil
}

// ChatStream performs streaming chat
func (c *RemoteAPIChat) ChatStream(ctx context.Context,
	messages []Message, opts *ChatOptions,
) (<-chan types.StreamResponse, error) {
	// Build request parameters
	req := c.buildChatCompletionRequest(messages, opts, true)

	// Create streaming response channel
	streamChan := make(chan types.StreamResponse)

	// Start streaming request
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		close(streamChan)
		return nil, fmt.Errorf("create chat completion stream: %w", err)
	}

	// Process streaming responses in the background
	go func() {
		defer close(streamChan)
		defer stream.Close()

		toolCallMap := make(map[int]*types.LLMToolCall)
		lastFunctionName := make(map[int]string)
		nameNotified := make(map[int]bool)

		buildOrderedToolCalls := func() []types.LLMToolCall {
			if len(toolCallMap) == 0 {
				return nil
			}
			result := make([]types.LLMToolCall, 0, len(toolCallMap))
			for i := 0; i < len(toolCallMap); i++ {
				if tc, ok := toolCallMap[i]; ok && tc != nil {
					result = append(result, *tc)
				}
			}
			if len(result) == 0 {
				return nil
			}
			return result
		}

		for {
			response, err := stream.Recv()
			if err != nil {
				// Send the final response, including collected tool calls
				streamChan <- types.StreamResponse{
					ResponseType: types.ResponseTypeAnswer,
					Content:      "",
					Done:         true,
					ToolCalls:    buildOrderedToolCalls(),
				}
				return
			}

			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				isDone := string(response.Choices[0].FinishReason) != ""

				// Collect tool calls (tool calls may be returned in multiple parts in streaming responses)
				if len(delta.ToolCalls) > 0 {
					for _, tc := range delta.ToolCalls {
						// Check if the tool call already exists (by index)
						var toolCallIndex int
						if tc.Index != nil {
							toolCallIndex = *tc.Index
						}
						toolCallEntry, exists := toolCallMap[toolCallIndex]
						if !exists || toolCallEntry == nil {
							toolCallEntry = &types.LLMToolCall{
								Type: string(tc.Type),
								Function: types.FunctionCall{
									Name:      "",
									Arguments: "",
								},
							}
							toolCallMap[toolCallIndex] = toolCallEntry
						}

						// Update ID and type
						if tc.ID != "" {
							toolCallEntry.ID = tc.ID
						}
						if tc.Type != "" {
							toolCallEntry.Type = string(tc.Type)
						}

						// Accumulate function name (may be returned in multiple parts)
						if tc.Function.Name != "" {
							toolCallEntry.Function.Name += tc.Function.Name
						}

						// Accumulate arguments (may be partial JSON)
						argsUpdated := false
						if tc.Function.Arguments != "" {
							toolCallEntry.Function.Arguments += tc.Function.Arguments
							argsUpdated = true
						}

						currName := toolCallEntry.Function.Name
						if currName != "" &&
							currName == lastFunctionName[toolCallIndex] &&
							argsUpdated &&
							!nameNotified[toolCallIndex] &&
							toolCallEntry.ID != "" {
							streamChan <- types.StreamResponse{
								ResponseType: types.ResponseTypeToolCall,
								Content:      "",
								Done:         false,
								Data: map[string]interface{}{
									"tool_name":    currName,
									"tool_call_id": toolCallEntry.ID,
								},
							}
							nameNotified[toolCallIndex] = true
						}

						lastFunctionName[toolCallIndex] = currName
					}
				}

				// Send content chunks
				if delta.Content != "" {
					streamChan <- types.StreamResponse{
						ResponseType: types.ResponseTypeAnswer,
						Content:      delta.Content,
						Done:         isDone,
						ToolCalls:    buildOrderedToolCalls(),
					}
				}

				// If this is the last response, ensure to send a response containing all tool calls
				if isDone && len(toolCallMap) > 0 {
					streamChan <- types.StreamResponse{
						ResponseType: types.ResponseTypeAnswer,
						Content:      "",
						Done:         true,
						ToolCalls:    buildOrderedToolCalls(),
					}
				}
			}
		}
	}()

	return streamChan, nil
}

// GetModelName gets the model name
func (c *RemoteAPIChat) GetModelName() string {
	return c.modelName
}

// GetModelID gets the model ID
func (c *RemoteAPIChat) GetModelID() string {
	return c.modelID
}
