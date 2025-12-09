package amdm

import (
	"time"
)

// LyricsResult represents the extracted lyrics result
type LyricsResult struct {
	URL       string    `json:"url"`
	Text      string    `json:"text"`
	FetchedAt time.Time `json:"fetched_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// SectionType represents different song sections
type SectionType string

const (
	SectionVerse  SectionType = "Куплет"
	SectionChorus SectionType = "Припев"
	SectionBridge SectionType = "Переход"
	SectionIntro  SectionType = "Вступление"
	SectionSolo   SectionType = "Проигрыш"
	SectionOutro  SectionType = "Кода"
)

// ProcessingConfig holds configuration for text processing
type ProcessingConfig struct {
	AllowedSections  []SectionType
	UnwantedSections []SectionType
	MaxLineBreaks    int
	PreserveSpacing  bool
}
