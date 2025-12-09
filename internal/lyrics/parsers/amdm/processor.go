package amdm

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// processHtmlContent processes the HTML content exactly like the TypeScript version
func (p *Parser) processHtmlContent(originalHtml string) string {
	// Replace chord elements with double line breaks instead of just removing them
	// Original: /<div[^>]*class="podbor__chord"[^>]*>.*?<\/div>/g, '\n\n'
	chordRegex := regexp.MustCompile(`<div[^>]*class="podbor__chord"[^>]*>.*?</div>`)
	processedHtml := chordRegex.ReplaceAllString(originalHtml, "\n\n")

	// Remove author comments completely
	// Original: /<span[^>]*class="podbor__author-comment"[^>]*>.*?<\/span>/g, ''
	authorCommentRegex := regexp.MustCompile(`<span[^>]*class="podbor__author-comment"[^>]*>.*?</span>`)
	processedHtml = authorCommentRegex.ReplaceAllString(processedHtml, "")

	// Remove any remaining comment text
	// Original: /\/\*[^*]*\*\//g, ''
	commentRegex1 := regexp.MustCompile(`/\*[^*]*\*\/`)
	processedHtml = commentRegex1.ReplaceAllString(processedHtml, "")

	// Remove incomplete comments at end of line
	// Original: /\/\*.*$/g, ''
	commentRegex2 := regexp.MustCompile(`/\*.*$`)
	processedHtml = commentRegex2.ReplaceAllString(processedHtml, "")

	// Remove other unwanted section elements (Вступление, Проигрыш, Кода)
	// Original: /<div[^>]*class="podbor__keyword"[^>]*>\s*\[(Вступление|Проигрыш|Кода)\][^<]*<\/div>/g, '\n\n'
	unwantedSectionRegex := regexp.MustCompile(`<div[^>]*class="podbor__keyword"[^>]*>\s*\[(Вступление|Проигрыш|Кода)\][^<]*</div>`)
	processedHtml = unwantedSectionRegex.ReplaceAllString(processedHtml, "\n\n")

	// Also remove plain text versions
	// Original: /\s*\[(Вступление|Проигрыш|Кода)\][^<]*/g, '\n\n'
	unwantedTextRegex := regexp.MustCompile(`\s*\[(Вступление|Проигрыш|Кода)\][^<]*`)
	processedHtml = unwantedTextRegex.ReplaceAllString(processedHtml, "\n\n")

	// Load the processed HTML with goquery
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(processedHtml))
	cleanText := doc.Text()

	// Process the text line by line (exact replication of TypeScript logic)
	return p.processTextLines(cleanText)
}
