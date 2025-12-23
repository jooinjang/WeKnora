package event

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// EventType represents the type of event in the system
type EventType string

const (
	// Query processing events
	EventQueryReceived   EventType = "query.received"
	EventQueryValidated  EventType = "query.validated"
	EventQueryPreprocess EventType = "query.preprocess"
	EventQueryRewrite    EventType = "query.rewrite"
	EventQueryRewritten  EventType = "query.rewritten"

	// Retrieval events
	EventRetrievalStart    EventType = "retrieval.start"
	EventRetrievalVector   EventType = "retrieval.vector"
	EventRetrievalKeyword  EventType = "retrieval.keyword"
	EventRetrievalEntity   EventType = "retrieval.entity"
	EventRetrievalComplete EventType = "retrieval.complete"

	// Rerank events
	EventRerankStart    EventType = "rerank.start"
	EventRerankComplete EventType = "rerank.complete"

	// Merge events
	EventMergeStart    EventType = "merge.start"
	EventMergeComplete EventType = "merge.complete"

	// Chat completion events
	EventChatStart    EventType = "chat.start"
	EventChatComplete EventType = "chat.complete"
	EventChatStream   EventType = "chat.stream"

	// Agent events
	EventAgentQuery    EventType = "agent.query"
	EventAgentPlan     EventType = "agent.plan"
	EventAgentStep     EventType = "agent.step"
	EventAgentTool     EventType = "agent.tool"
	EventAgentComplete EventType = "agent.complete"

	// Agent streaming events (for real-time feedback)
	EventAgentThought     EventType = "thought"
	EventAgentToolCall    EventType = "tool_call"
	EventAgentToolResult  EventType = "tool_result"
	EventAgentReflection  EventType = "reflection"
	EventAgentReferences  EventType = "references"
	EventAgentFinalAnswer EventType = "final_answer"

	// Error events
	EventError EventType = "error"

	// Session events
	EventSessionTitle EventType = "session_title"

	// Control events
	EventStop EventType = "stop"
)

// Event represents an event in the system
type Event struct {
	ID        string
	Type      EventType
	SessionID string
	Data      interface{}
	Metadata  map[string]interface{}
	RequestID string
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// EventBus manages event publishing and subscription
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[EventType][]EventHandler
	asyncMode bool
}

// NewEventBus creates a new EventBus instance
func NewEventBus() *EventBus {
	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		asyncMode: false,
	}
}

// NewAsyncEventBus creates a new EventBus with async mode enabled
func NewAsyncEventBus() *EventBus {
	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		asyncMode: true,
	}
}

// On registers an event handler for a specific event type
// Multiple handlers can be registered for the same event type
func (eb *EventBus) On(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Off removes all handlers for a specific event type
func (eb *EventBus) Off(eventType EventType) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, eventType)
}

// Emit publishes an event to all registered handlers
// Returns error if any handler fails (in sync mode)
// Automatically generates an ID for the event if not provided (from source)
func (eb *EventBus) Emit(ctx context.Context, event Event) error {
	// Auto-generate ID if not provided (from source)
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		// No handlers registered for this event type
		return nil
	}

	if eb.asyncMode {
		// Async mode: fire and forget
		for _, handler := range handlers {
			h := handler // capture loop variable
			go func() {
				_ = h(ctx, event)
			}()
		}
		return nil
	}

	// Sync mode: execute handlers sequentially
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("event handler failed for %s: %w", event.Type, err)
		}
	}

	return nil
}

// EmitAndWait publishes an event and waits for all handlers to complete
// This method works in both sync and async mode
// Automatically generates an ID for the event if not provided (from source)
func (eb *EventBus) EmitAndWait(ctx context.Context, event Event) error {
	// Auto-generate ID if not provided (from source)
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		h := handler // capture loop variable

		go func() {
			defer wg.Done()
			if err := h(ctx, event); err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	for err := range errChan {
		if err != nil {
			return fmt.Errorf("event handler failed for %s: %w", event.Type, err)
		}
	}

	return nil
}

// HasHandlers checks if there are any handlers registered for an event type
func (eb *EventBus) HasHandlers(eventType EventType) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, exists := eb.handlers[eventType]
	return exists && len(handlers) > 0
}

// GetHandlerCount returns the number of handlers for a specific event type
func (eb *EventBus) GetHandlerCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if handlers, exists := eb.handlers[eventType]; exists {
		return len(handlers)
	}
	return 0
}

// Clear removes all event handlers
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers = make(map[EventType][]EventHandler)
}
