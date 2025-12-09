package amdm

import (
	"regexp"
	"strings"
)

// finalCleanup applies the exact final cleanup from the TypeScript version
func (p *Parser) finalCleanup(lyrics string) string {
	// Final cleanup - ensure unwanted sections are replaced with 2 line breaks
	// Original: /\s*\[(Вступление|Проигрыш|Кода)\][^<]*/g, '\n\n'
	unwantedFinalRegex := regexp.MustCompile(`\s*\[(Вступление|Проигрыш|Кода)\][^<]*`)
	lyrics = unwantedFinalRegex.ReplaceAllString(lyrics, "\n\n")

	// Specific fixes for exact cases
	lyrics = strings.ReplaceAll(lyrics, "[Проигрыш]:", "\n\n")
	lyrics = strings.ReplaceAll(lyrics, "[Вступление]:", "\n\n")
	lyrics = strings.ReplaceAll(lyrics, "[Кода]:", "\n\n")

	// Specific fix for the exact location where [Проигрыш]: should be
	// Original: /а к одиннадцати туз\.\s*Там под окном Зе-Ка\./g
	specificFixRegex := regexp.MustCompile(`а к одиннадцати туз\.\s*Там под окном Зе-Ка\.`)
	lyrics = specificFixRegex.ReplaceAllString(lyrics, "а к одиннадцати туз.\n\nТам под окном Зе-Ка.")

	// Normalize excessive consecutive line breaks but preserve intentional double breaks
	// Original: /\n{4,}/g, '\n\n\n' (Allow up to 3 consecutive line breaks)
	excessiveBreaksRegex := regexp.MustCompile(`\n{4,}`)
	lyrics = excessiveBreaksRegex.ReplaceAllString(lyrics, "\n\n\n")

	return strings.TrimSpace(lyrics)
}
