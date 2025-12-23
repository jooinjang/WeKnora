package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// FAQChunkMetadata defines the structure of FAQ entries in Chunk.Metadata
type FAQChunkMetadata struct {
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions,omitempty"`
	NegativeQuestions []string       `json:"negative_questions,omitempty"`
	Answers           []string       `json:"answers,omitempty"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy,omitempty"`
	Version           int            `json:"version,omitempty"`
	Source            string         `json:"source,omitempty"`
}

// GeneratedQuestion represents a single AI-generated question
type GeneratedQuestion struct {
	ID       string `json:"id"`       // Unique identifier, used to construct source_id
	Question string `json:"question"` // Question content
}

// DocumentChunkMetadata defines the metadata structure of document chunks
// Used to store enhanced information such as AI-generated questions
type DocumentChunkMetadata struct {
	// GeneratedQuestions stores AI-generated related questions for this chunk
	// These questions are independently indexed to improve recall rate
	GeneratedQuestions []GeneratedQuestion `json:"generated_questions,omitempty"`
}

// GetQuestionStrings returns a list of question content strings (compatible with legacy code)
func (m *DocumentChunkMetadata) GetQuestionStrings() []string {
	if m == nil || len(m.GeneratedQuestions) == 0 {
		return nil
	}
	result := make([]string, len(m.GeneratedQuestions))
	for i, q := range m.GeneratedQuestions {
		result[i] = q.Question
	}
	return result
}

// DocumentMetadata parses document metadata from Chunk
func (c *Chunk) DocumentMetadata() (*DocumentChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta DocumentChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// SetDocumentMetadata sets the document metadata for Chunk
func (c *Chunk) SetDocumentMetadata(meta *DocumentChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		return nil
	}
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	return nil
}

// Normalize cleans up whitespace and duplicate items
func (m *FAQChunkMetadata) Normalize() {
	if m == nil {
		return
	}
	m.StandardQuestion = strings.TrimSpace(m.StandardQuestion)
	m.SimilarQuestions = normalizeStrings(m.SimilarQuestions)
	m.NegativeQuestions = normalizeStrings(m.NegativeQuestions)
	m.Answers = normalizeStrings(m.Answers)
	if m.Version <= 0 {
		m.Version = 1
	}
}

// FAQMetadata parses FAQ metadata from Chunk
func (c *Chunk) FAQMetadata() (*FAQChunkMetadata, error) {
	if c == nil || len(c.Metadata) == 0 {
		return nil, nil
	}
	var meta FAQChunkMetadata
	if err := json.Unmarshal(c.Metadata, &meta); err != nil {
		return nil, err
	}
	meta.Normalize()
	return &meta, nil
}

// SetFAQMetadata sets FAQ metadata for Chunk
func (c *Chunk) SetFAQMetadata(meta *FAQChunkMetadata) error {
	if c == nil {
		return nil
	}
	if meta == nil {
		c.Metadata = nil
		c.ContentHash = ""
		return nil
	}
	meta.Normalize()
	bytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.Metadata = JSON(bytes)
	// Calculate and set ContentHash
	c.ContentHash = CalculateFAQContentHash(meta)
	return nil
}

// CalculateFAQContentHash calculates the hash value of FAQ content
// Hash is based on: standard question + similar questions (sorted) + negative examples (sorted) + answers (sorted)
// Used for fast matching and deduplication
func CalculateFAQContentHash(meta *FAQChunkMetadata) string {
	if meta == nil {
		return ""
	}

	// Create a copy and normalize
	normalized := *meta
	normalized.Normalize()

	// Sort arrays (ensure same content produces same hash)
	similarQuestions := make([]string, len(normalized.SimilarQuestions))
	copy(similarQuestions, normalized.SimilarQuestions)
	sort.Strings(similarQuestions)

	negativeQuestions := make([]string, len(normalized.NegativeQuestions))
	copy(negativeQuestions, normalized.NegativeQuestions)
	sort.Strings(negativeQuestions)

	answers := make([]string, len(normalized.Answers))
	copy(answers, normalized.Answers)
	sort.Strings(answers)

	// Build string for hashing: standard question + similar questions + negative examples + answers
	var builder strings.Builder
	builder.WriteString(normalized.StandardQuestion)
	builder.WriteString("|")
	builder.WriteString(strings.Join(similarQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(negativeQuestions, ","))
	builder.WriteString("|")
	builder.WriteString(strings.Join(answers, ","))

	// Calculate SHA256 hash
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// AnswerStrategy defines the answer return strategy
type AnswerStrategy string

const (
	// AnswerStrategyAll returns all answers
	AnswerStrategyAll AnswerStrategy = "all"
	// AnswerStrategyRandom randomly returns one answer
	AnswerStrategyRandom AnswerStrategy = "random"
)

// FAQEntry represents an FAQ entry returned to the frontend
type FAQEntry struct {
	ID                string         `json:"id"`
	ChunkID           string         `json:"chunk_id"`
	KnowledgeID       string         `json:"knowledge_id"`
	KnowledgeBaseID   string         `json:"knowledge_base_id"`
	TagID             string         `json:"tag_id"`
	IsEnabled         bool           `json:"is_enabled"`
	IsRecommended     bool           `json:"is_recommended"`
	StandardQuestion  string         `json:"standard_question"`
	SimilarQuestions  []string       `json:"similar_questions"`
	NegativeQuestions []string       `json:"negative_questions"`
	Answers           []string       `json:"answers"`
	AnswerStrategy    AnswerStrategy `json:"answer_strategy"`
	IndexMode         FAQIndexMode   `json:"index_mode"`
	UpdatedAt         time.Time      `json:"updated_at"`
	CreatedAt         time.Time      `json:"created_at"`
	Score             float64        `json:"score,omitempty"`
	MatchType         MatchType      `json:"match_type,omitempty"`
	ChunkType         ChunkType      `json:"chunk_type"`
}

// FAQEntryPayload is the payload used for creating/updating FAQ entries
type FAQEntryPayload struct {
	StandardQuestion  string          `json:"standard_question"    binding:"required"`
	SimilarQuestions  []string        `json:"similar_questions"`
	NegativeQuestions []string        `json:"negative_questions"`
	Answers           []string        `json:"answers"              binding:"required"`
	AnswerStrategy    *AnswerStrategy `json:"answer_strategy,omitempty"`
	TagID             string          `json:"tag_id"`
	TagName           string          `json:"tag_name"`
	IsEnabled         *bool           `json:"is_enabled,omitempty"`
	IsRecommended     *bool           `json:"is_recommended,omitempty"`
}

const (
	FAQBatchModeAppend  = "append"
	FAQBatchModeReplace = "replace"
)

// FAQBatchUpsertPayload is used for batch importing FAQ entries
type FAQBatchUpsertPayload struct {
	Entries     []FAQEntryPayload `json:"entries"      binding:"required"`
	Mode        string            `json:"mode"         binding:"oneof=append replace"`
	KnowledgeID string            `json:"knowledge_id"`
}

// FAQSearchRequest defines FAQ search request parameters
type FAQSearchRequest struct {
	QueryText       string  `json:"query_text"       binding:"required"`
	VectorThreshold float64 `json:"vector_threshold"`
	MatchCount      int     `json:"match_count"`
}

// FAQEntryFieldsUpdate defines field updates for a single FAQ entry
type FAQEntryFieldsUpdate struct {
	IsEnabled     *bool   `json:"is_enabled,omitempty"`
	IsRecommended *bool   `json:"is_recommended,omitempty"`
	TagID         *string `json:"tag_id,omitempty"`
	// Can be extended with more fields in the future
}

// FAQEntryFieldsBatchUpdate is the request for batch updating FAQ entry fields
// Supports two modes:
// 1. Update by entry ID: use the ByID field
// 2. Update by Tag: use the ByTag field, applying the same update to all entries under that Tag
type FAQEntryFieldsBatchUpdate struct {
	// ByID updates by entry ID, key is the entry ID
	ByID map[string]FAQEntryFieldsUpdate `json:"by_id,omitempty"`
	// ByTag updates by Tag in batch, key is TagID (empty string means uncategorized)
	ByTag map[string]FAQEntryFieldsUpdate `json:"by_tag,omitempty"`
}

// FAQImportTaskStatus represents the import task status
type FAQImportTaskStatus string

const (
	// FAQImportStatusPending represents the pending status of the FAQ import task
	FAQImportStatusPending FAQImportTaskStatus = "pending"
	// FAQImportStatusProcessing represents the processing status of the FAQ import task
	FAQImportStatusProcessing FAQImportTaskStatus = "processing"
	// FAQImportStatusCompleted represents the completed status of the FAQ import task
	FAQImportStatusCompleted FAQImportTaskStatus = "completed"
	// FAQImportStatusFailed represents the failed status of the FAQ import task
	FAQImportStatusFailed FAQImportTaskStatus = "failed"
)

// FAQImportProgress represents the progress of an FAQ import task stored in Redis
type FAQImportProgress struct {
	TaskID      string              `json:"task_id"`       // UUID for the import task
	KBID        string              `json:"kb_id"`         // Knowledge Base ID
	KnowledgeID string              `json:"knowledge_id"`  // FAQ Knowledge ID
	Status      FAQImportTaskStatus `json:"status"`        // Task status
	Progress    int                 `json:"progress"`      // 0-100 percentage
	Total       int                 `json:"total"`         // Total entries to import
	Processed   int                 `json:"processed"`     // Entries processed so far
	Message     string              `json:"message"`       // Status message
	Error       string              `json:"error"`         // Error message if failed
	CreatedAt   int64               `json:"created_at"`    // Task creation timestamp
	UpdatedAt   int64               `json:"updated_at"`    // Last update timestamp
}

// FAQImportMetadata stores FAQ import task information in Knowledge.Metadata
// Deprecated: Use FAQImportProgress with Redis storage instead
type FAQImportMetadata struct {
	ImportProgress  int `json:"import_progress"` // 0-100
	ImportTotal     int `json:"import_total"`
	ImportProcessed int `json:"import_processed"`
}

// ToJSON converts the metadata to JSON type.
func (m *FAQImportMetadata) ToJSON() (JSON, error) {
	if m == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return JSON(bytes), nil
}

// ParseFAQImportMetadata parses FAQ import metadata from Knowledge.
func ParseFAQImportMetadata(k *Knowledge) (*FAQImportMetadata, error) {
	if k == nil || len(k.Metadata) == 0 {
		return nil, nil
	}
	var metadata FAQImportMetadata
	if err := json.Unmarshal(k.Metadata, &metadata); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	dedup := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		dedup = append(dedup, trimmed)
	}
	if len(dedup) == 0 {
		return nil
	}
	return dedup
}
