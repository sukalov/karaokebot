package lyrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/sukalov/karaokebot/internal/logger"
	"github.com/sukalov/karaokebot/internal/lyrics/parsers/amdm"
)

// LyricsResult represents the result of lyrics extraction
type LyricsResult struct {
	URL       string    `json:"url"`
	Text      string    `json:"text"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Service handles lyrics extraction for different sources
type Service struct {
	amdmParser *amdm.Parser
}

// NewService creates a new lyrics service
func NewService() *Service {
	return &Service{
		amdmParser: amdm.NewParser(),
	}
}

// ExtractLyrics extracts lyrics from a URL based on the source
func (s *Service) ExtractLyrics(url string) (*LyricsResult, error) {
	logger.Debug(fmt.Sprintf("ExtractLyrics called with URL: %s", url))

	// Detect source based on URL pattern
	if strings.Contains(url, "amdm.ru") {
		logger.Debug("Detected amdm.ru URL, using AMDm parser")
		return s.extractFromAmdm(url)
	}

	// Add other parsers here as needed
	logger.Error(fmt.Sprintf("Unsupported URL source: %s", url))
	return nil, fmt.Errorf("unsupported URL source: %s", url)
}

// extractFromAmdm extracts lyrics from AmDm.ru
func (s *Service) extractFromAmdm(url string) (*LyricsResult, error) {
	logger.Debug(fmt.Sprintf("extractFromAmdm called with URL: %s", url))

	result, err := s.amdmParser.ExtractLyricsFromAmdm(url)
	if err != nil {
		logger.Error(fmt.Sprintf("amdmParser.ExtractLyricsFromAmdm failed for URL: %s\nError: %v", url, err))
		return nil, err
	}

	logger.Debug(fmt.Sprintf("extractFromAmdm succeeded for URL: %s\nLyrics length: %d chars\nSuccess: %v",
		url, len(result.Text), result.Success))

	return &LyricsResult{
		URL:       result.URL,
		Text:      result.Text,
		Source:    "amdm.ru",
		FetchedAt: result.FetchedAt,
	}, nil
}
