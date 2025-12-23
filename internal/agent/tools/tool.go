package tools

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	description string
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string) BaseTool {
	return BaseTool{
		name:        name,
		description: description,
	}
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// ToolExecutor is a helper interface for executing tools
type ToolExecutor interface {
	types.Tool

	// GetContext returns any context-specific data needed for tool execution
	GetContext() map[string]interface{}
}

// Shared helper functions for tool output formatting

// GetRelevanceLevel converts a score to a human-readable relevance level
func GetRelevanceLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "High relevance"
	case score >= 0.6:
		return "Medium relevance"
	case score >= 0.4:
		return "Low relevance"
	default:
		return "Weak relevance"
	}
}

// FormatMatchType converts MatchType to a human-readable string
func FormatMatchType(mt types.MatchType) string {
	switch mt {
	case types.MatchTypeEmbedding:
		return "Vector matching"
	case types.MatchTypeKeywords:
		return "Keyword matching"
	case types.MatchTypeNearByChunk:
		return "Adjacent chunk matching"
	case types.MatchTypeHistory:
		return "History matching"
	case types.MatchTypeParentChunk:
		return "Parent chunk matching"
	case types.MatchTypeRelationChunk:
		return "Relation chunk matching"
	case types.MatchTypeGraph:
		return "Graph matching"
	default:
		return fmt.Sprintf("Unknown type(%d)", mt)
	}
}
