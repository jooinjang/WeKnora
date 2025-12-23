// Package types defines data structures and types used throughout the system
// These types are shared across different service modules to ensure data consistency
package types

import (
	"time"

	"gorm.io/gorm"
)

// ChunkType defines different types of Chunks
type ChunkType string

const (
	// ChunkTypeText represents a regular text Chunk
	ChunkTypeText ChunkType = "text"
	// ChunkTypeImageOCR represents a Chunk of image OCR text
	ChunkTypeImageOCR ChunkType = "image_ocr"
	// ChunkTypeImageCaption represents a Chunk of image description
	ChunkTypeImageCaption ChunkType = "image_caption"
	// ChunkTypeSummary represents a summary type Chunk
	ChunkTypeSummary = "summary"
	// ChunkTypeEntity represents an entity type Chunk
	ChunkTypeEntity ChunkType = "entity"
	// ChunkTypeRelationship represents a relationship type Chunk
	ChunkTypeRelationship ChunkType = "relationship"
	// ChunkTypeFAQ represents an FAQ entry Chunk
	ChunkTypeFAQ ChunkType = "faq"
	// ChunkTypeWebSearch represents a web search result Chunk
	ChunkTypeWebSearch ChunkType = "web_search"
)

// ChunkStatus defines different states of Chunks
type ChunkStatus int

const (
	ChunkStatusDefault ChunkStatus = 0
	// ChunkStatusStored represents a stored Chunk
	ChunkStatusStored ChunkStatus = 1
	// ChunkStatusIndexed represents an indexed Chunk
	ChunkStatusIndexed ChunkStatus = 2
)

// ChunkFlags defines flag bits for Chunks, used to manage multiple boolean states
type ChunkFlags int

const (
	// ChunkFlagRecommended represents the recommendable state (1 << 0 = 1)
	// When this flag is set, the Chunk can be recommended to users
	ChunkFlagRecommended ChunkFlags = 1 << 0
	// Future extensible flags:
	// ChunkFlagPinned ChunkFlags = 1 << 1  // Pinned
	// ChunkFlagHot    ChunkFlags = 1 << 2  // Hot
)

// HasFlag checks if the specified flag is set
func (f ChunkFlags) HasFlag(flag ChunkFlags) bool {
	return f&flag != 0
}

// SetFlag sets the specified flag
func (f ChunkFlags) SetFlag(flag ChunkFlags) ChunkFlags {
	return f | flag
}

// ClearFlag clears the specified flag
func (f ChunkFlags) ClearFlag(flag ChunkFlags) ChunkFlags {
	return f &^ flag
}

// ToggleFlag toggles the specified flag
func (f ChunkFlags) ToggleFlag(flag ChunkFlags) ChunkFlags {
	return f ^ flag
}

// ImageInfo represents image information associated with a Chunk
type ImageInfo struct {
	// Image URL (COS)
	URL string `json:"url"          gorm:"type:text"`
	// Original image URL
	OriginalURL string `json:"original_url" gorm:"type:text"`
	// Start position of the image in the text
	StartPos int `json:"start_pos"`
	// End position of the image in the text
	EndPos int `json:"end_pos"`
	// Image caption
	Caption string `json:"caption"`
	// Image OCR text
	OCRText string `json:"ocr_text"`
}

// Chunk represents a document chunk
// Chunks are meaningful text segments extracted from original documents
// and are the basic units of knowledge base retrieval
// Each chunk contains a portion of the original content
// and maintains its positional relationship with the original text
// Chunks can be independently embedded as vectors and retrieved, supporting precise content localization
type Chunk struct {
	// Unique identifier of the chunk, using UUID format
	ID string `json:"id"                       gorm:"type:varchar(36);primaryKey"`
	// Tenant ID, used for multi-tenant isolation
	TenantID uint64 `json:"tenant_id"`
	// ID of the parent knowledge, associated with the Knowledge model
	KnowledgeID string `json:"knowledge_id"`
	// ID of the knowledge base, for quick location
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// Optional tag ID for categorization within a knowledge base (used for FAQ)
	TagID string `json:"tag_id"                   gorm:"type:varchar(36);index"`
	// Actual text content of the chunk
	Content string `json:"content"`
	// Index position of the chunk in the original document
	ChunkIndex int `json:"chunk_index"`
	// Whether the chunk is enabled, can be used to temporarily disable certain chunks
	IsEnabled bool `json:"is_enabled"               gorm:"default:true"`
	// Flags stores bit flags for multiple boolean states (such as recommendation status, etc.)
	// Default value is ChunkFlagRecommended (1), indicating default recommendable
	Flags ChunkFlags `json:"flags"                    gorm:"default:1"`
	// Status of the chunk
	Status int `json:"status"                   gorm:"default:0"`
	// Starting character position in the original text
	StartAt int `json:"start_at"`
	// Ending character position in the original text
	EndAt int `json:"end_at"`
	// Previous chunk ID
	PreChunkID string `json:"pre_chunk_id"`
	// Next chunk ID
	NextChunkID string `json:"next_chunk_id"`
	// Chunk type, used to distinguish different types of Chunks
	ChunkType ChunkType `json:"chunk_type"               gorm:"type:varchar(20);default:'text'"`
	// Parent Chunk ID, used to associate image Chunks with original text Chunks
	ParentChunkID string `json:"parent_chunk_id"          gorm:"type:varchar(36);index"`
	// Relationship Chunk IDs, used to associate relationship Chunks with original text Chunks
	RelationChunks JSON `json:"relation_chunks"          gorm:"type:json"`
	// Indirect relationship Chunk IDs, used to associate indirect relationship Chunks with original text Chunks
	IndirectRelationChunks JSON `json:"indirect_relation_chunks" gorm:"type:json"`
	// Metadata stores chunk-level extended information, such as FAQ metadata
	Metadata JSON `json:"metadata"                 gorm:"type:json"`
	// ContentHash stores the hash value of content, used for fast matching (mainly for FAQ)
	ContentHash string `json:"content_hash"             gorm:"type:varchar(64);index"`
	// Image information, stored as JSON
	ImageInfo string `json:"image_info"               gorm:"type:text"`
	// Chunk creation time
	CreatedAt time.Time `json:"created_at"`
	// Chunk last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Soft delete marker, supports data recovery
	DeletedAt gorm.DeletedAt `json:"deleted_at"               gorm:"index"`
}
