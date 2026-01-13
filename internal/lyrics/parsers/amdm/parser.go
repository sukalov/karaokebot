package amdm

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sukalov/karaokebot/internal/logger"
)

// Parser handles the HTML parsing and lyrics extraction
type Parser struct {
	client *Client
	config *ProcessingConfig
}

// NewParser creates a new AmDm parser
func NewParser() *Parser {
	config := &ProcessingConfig{
		AllowedSections:  []SectionType{SectionVerse, SectionChorus, SectionBridge},
		UnwantedSections: []SectionType{SectionIntro, SectionSolo, SectionOutro},
		MaxLineBreaks:    3,
		PreserveSpacing:  true,
	}

	return &Parser{
		client: NewClient(),
		config: config,
	}
}

// ExtractLyricsFromAmdm extracts lyrics from an AmDm.ru page
func (p *Parser) ExtractLyricsFromAmdm(url string) (*LyricsResult, error) {
	fmt.Printf("Fetching page: %s\n", url)
	logger.Debug(fmt.Sprintf("ExtractLyricsFromAmdm: Fetching page %s", url))

	html, err := p.client.FetchPage(url)
	if err != nil {
		logger.Error(fmt.Sprintf("ExtractLyricsFromAmdm: Failed to fetch page %s\nError: %v", url, err))
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   err.Error(),
		}, err
	}

	logger.Debug(fmt.Sprintf("ExtractLyricsFromAmdm: Successfully fetched page %s (HTML length: %d chars)", url, len(html)))

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		logger.Error(fmt.Sprintf("ExtractLyricsFromAmdm: Failed to parse HTML for %s\nError: %v", url, err))
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   fmt.Sprintf("failed to parse HTML: %v", err),
		}, err
	}

	// Find the target element: <pre itemprop="chordsBlock" class="field__podbor_new podbor__text">
	selection := doc.Find(`pre[itemprop="chordsBlock"].field__podbor_new.podbor__text`)
	if selection.Length() == 0 {
		logger.Error(fmt.Sprintf("ExtractLyricsFromAmdm: Target element not found for URL %s\nSearched for: pre[itemprop=\"chordsBlock\"].field__podbor_new.podbor__text", url))
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   "Could not find target element with chords and lyrics",
		}, fmt.Errorf("target element not found")
	}

	fmt.Println("Found target element, extracting content...")
	logger.Success(fmt.Sprintf("ExtractLyricsFromAmdm: Found target element for URL %s", url))

	// Get original HTML content
	originalHtml, _ := selection.Html()

	logger.Debug(fmt.Sprintf("ExtractLyricsFromAmdm: Extracted original HTML content (length: %d chars)", len(originalHtml)))

	// Process the HTML content
	processedText := p.processHtmlContent(originalHtml)

	logger.Debug(fmt.Sprintf("ExtractLyricsFromAmdm: Processed lyrics text (length: %d chars)", len(processedText)))

	return &LyricsResult{
		URL:       url,
		Text:      processedText,
		FetchedAt: time.Now(),
		Success:   true,
	}, nil
}
