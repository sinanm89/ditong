// Package normalizer handles multi-language character normalization to ASCII.
package normalizer

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// charMap maps non-ASCII characters to ASCII equivalents.
var charMap = map[rune]string{
	// Turkish
	'ç': "c", 'Ç': "c",
	'ş': "s", 'Ş': "s",
	'ğ': "g", 'Ğ': "g",
	'ı': "i", 'İ': "i",
	// German
	'ä': "a", 'Ä': "a",
	'ö': "o", 'Ö': "o",
	'ü': "u", 'Ü': "u",
	'ß': "ss",
	// French
	'à': "a", 'â': "a", 'æ': "ae",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
	'î': "i", 'ï': "i",
	'ô': "o", 'œ': "oe",
	'ù': "u", 'û': "u",
	'ÿ': "y",
	// Spanish
	'á': "a", 'í': "i", 'ó': "o", 'ú': "u",
	'ñ': "n", 'Ñ': "n",
	// Portuguese
	'ã': "a", 'õ': "o",
	// Polish
	'ą': "a", 'ć': "c", 'ę': "e", 'ł': "l",
	'ń': "n", 'ś': "s", 'ź': "z", 'ż': "z",
	// Czech/Slovak
	'č': "c", 'ď': "d", 'ě': "e", 'ň': "n",
	'ř': "r", 'š': "s", 'ť': "t", 'ů': "u", 'ž': "z",
	// Nordic
	'å': "a", 'Å': "a",
	'ø': "o", 'Ø': "o",
	// Romanian
	'ă': "a", 'ț': "t", 'ș': "s",
}

var alphaPattern = regexp.MustCompile(`^[a-z]+$`)

// NormalizeChar normalizes a single character to ASCII equivalent.
func NormalizeChar(r rune) string {
	// Check direct mapping
	if ascii, ok := charMap[r]; ok {
		return ascii
	}

	// Check lowercase version
	lower := unicode.ToLower(r)
	if ascii, ok := charMap[lower]; ok {
		return ascii
	}

	// Fall back to Unicode decomposition
	normalized := norm.NFD.String(string(r))
	var result strings.Builder
	for _, c := range normalized {
		if !unicode.Is(unicode.Mn, c) { // Not a combining mark
			if c < 128 { // ASCII
				result.WriteRune(unicode.ToLower(c))
			}
		}
	}
	if result.Len() > 0 {
		return result.String()
	}

	return strings.ToLower(string(r))
}

// NormalizeWord normalizes a word to lowercase ASCII.
func NormalizeWord(word string) string {
	var result strings.Builder
	result.Grow(len(word))

	for _, r := range word {
		result.WriteString(NormalizeChar(r))
	}

	return result.String()
}

// IsValidIdentifier checks if normalized word is valid (ASCII a-z only).
func IsValidIdentifier(word string) bool {
	return alphaPattern.MatchString(word)
}

// NormalizeAndValidate normalizes word and returns it if valid, else empty string.
func NormalizeAndValidate(word string) string {
	normalized := NormalizeWord(word)
	if IsValidIdentifier(normalized) {
		return normalized
	}
	return ""
}
