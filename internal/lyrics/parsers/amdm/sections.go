package amdm

import (
	"regexp"
	"strings"
)

// processTextLines processes text line by line exactly like the TypeScript version
func (p *Parser) processTextLines(cleanText string) string {
	lines := strings.Split(cleanText, "\n")
	var processedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		// Handle section markers - only keep specified ones
		if strings.HasPrefix(trimmedLine, "[") {
			p.handleSectionMarker(trimmedLine, &processedLines)
			continue
		}

		// Skip lines that are just chord separators or empty
		// Original: /^[\s\|]*$/
		chordSeparatorRegex := regexp.MustCompile(`^[\s\|]*$`)
		if chordSeparatorRegex.MatchString(trimmedLine) {
			continue
		}

		// Remove any remaining comment artifacts and add text line
		// Original: /\/\*[^*]*\*?/g, ''
		commentArtifactRegex := regexp.MustCompile(`/\*[^*]*\*?`)
		cleanLine := commentArtifactRegex.ReplaceAllString(trimmedLine, "")
		cleanLine = strings.TrimSpace(cleanLine)

		// Remove any stray asterisks
		cleanLine = strings.ReplaceAll(cleanLine, "*", "")
		cleanLine = strings.TrimSpace(cleanLine)

		// Remove any stray slashes
		cleanLine = strings.ReplaceAll(cleanLine, "/", "")
		cleanLine = strings.TrimSpace(cleanLine)

		if cleanLine != "" {
			processedLines = append(processedLines, cleanLine)
		}
	}

	// Join all processed lines with single line breaks
	lyrics := strings.Join(processedLines, "\n")

	// Apply final cleanup (exact replication)
	return p.finalCleanup(lyrics)
}

// handleSectionMarker handles section markers exactly like the TypeScript version
func (p *Parser) handleSectionMarker(trimmedLine string, processedLines *[]string) {
	allowedSections := []string{"[Куплет]:", "[Припев]:", "[Переход]:"}
	unwantedSections := []string{"[Вступление]:", "[Проигрыш]:", "[Кода]:"}

	isAllowedSection := false
	for _, section := range allowedSections {
		if strings.Contains(trimmedLine, section) {
			isAllowedSection = true
			break
		}
	}

	isUnwantedSection := false
	for _, section := range unwantedSections {
		if strings.Contains(trimmedLine, section) {
			isUnwantedSection = true
			break
		}
	}

	if isAllowedSection {
		// Extract section name
		// Original: /\[([^\]]+):\]/
		sectionRegex := regexp.MustCompile(`\[([^\]]+):\]`)
		match := sectionRegex.FindStringSubmatch(trimmedLine)
		var sectionName string
		if len(match) > 1 {
			sectionName = "[" + match[1] + "]:"
		} else {
			sectionName = trimmedLine
		}

		// Add section marker with 2 line breaks before and 1 after
		*processedLines = append(*processedLines, "")          // First line break before
		*processedLines = append(*processedLines, sectionName) // Section marker
	} else if isUnwantedSection {
		// Replace unwanted sections with 2 line breaks
		*processedLines = append(*processedLines, "") // First line break
		*processedLines = append(*processedLines, "") // Second line break
	}
}
