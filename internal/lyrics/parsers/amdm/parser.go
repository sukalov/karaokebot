package amdm

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	html, err := p.client.FetchPage(url)
	if err != nil {
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   fmt.Sprintf("failed to parse HTML: %v", err),
		}, err
	}

	// Find the target element: <pre itemprop="chordsBlock" class="field__podbor_new podbor__text">
	selection := doc.Find(`pre[itemprop="chordsBlock"].field__podbor_new.podbor__text`)
	if selection.Length() == 0 {
		return &LyricsResult{
			URL:     url,
			Success: false,
			Error:   "Could not find target element with chords and lyrics",
		}, fmt.Errorf("target element not found")
	}

	// Get original HTML content
	originalHtml, _ := selection.Html()

	// Process the HTML content
	processedText := p.processHtmlContent(originalHtml)

	return &LyricsResult{
		URL:       url,
		Text:      processedText,
		FetchedAt: time.Now(),
		Success:   true,
	}, nil
}
