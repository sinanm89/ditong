// Package ipa provides IPA (International Phonetic Alphabet) transcription.
package ipa

import (
	"strings"
	"unicode"
)

// Transcriber handles IPA transcription for words.
type Transcriber struct {
	language string
	rules    map[string]string
}

// NewTranscriber creates a transcriber for the given language.
func NewTranscriber(language string) *Transcriber {
	t := &Transcriber{
		language: language,
		rules:    make(map[string]string),
	}
	t.loadRules()
	return t
}

// loadRules loads transcription rules for the configured language.
func (t *Transcriber) loadRules() {
	switch t.language {
	case "en":
		t.rules = englishRules
	case "tr":
		t.rules = turkishRules
	case "de":
		t.rules = germanRules
	case "fr":
		t.rules = frenchRules
	default:
		// Fallback to basic ASCII mapping
		t.rules = basicRules
	}
}

// Transcribe converts a word to IPA notation.
func (t *Transcriber) Transcribe(word string) string {
	word = strings.ToLower(word)
	var result strings.Builder

	i := 0
	for i < len(word) {
		matched := false

		// Try multi-character rules first (longest match)
		for length := 4; length > 0; length-- {
			if i+length <= len(word) {
				substr := word[i : i+length]
				if ipa, ok := t.rules[substr]; ok {
					result.WriteString(ipa)
					i += length
					matched = true
					break
				}
			}
		}

		if !matched {
			// Single character fallback
			ch := rune(word[i])
			if unicode.IsLetter(ch) {
				if ipa, ok := t.rules[string(ch)]; ok {
					result.WriteString(ipa)
				} else {
					result.WriteRune(ch)
				}
			}
			i++
		}
	}

	return result.String()
}

// Language returns the transcriber's language.
func (t *Transcriber) Language() string {
	return t.language
}

// Basic ASCII to IPA mapping (fallback)
var basicRules = map[string]string{
	"a": "a", "b": "b", "c": "k", "d": "d", "e": "e",
	"f": "f", "g": "g", "h": "h", "i": "i", "j": "dʒ",
	"k": "k", "l": "l", "m": "m", "n": "n", "o": "o",
	"p": "p", "q": "k", "r": "r", "s": "s", "t": "t",
	"u": "u", "v": "v", "w": "w", "x": "ks", "y": "j",
	"z": "z",
}

// English phonetic rules (simplified)
var englishRules = map[string]string{
	// Digraphs and common patterns
	"th":   "θ",
	"ch":   "tʃ",
	"sh":   "ʃ",
	"ph":   "f",
	"wh":   "w",
	"ng":   "ŋ",
	"ck":   "k",
	"gh":   "", // often silent
	"kn":   "n",
	"wr":   "r",
	"mb":   "m",
	"tion": "ʃən",
	"sion": "ʒən",
	"ough": "oʊ",
	"igh":  "aɪ",
	"eigh": "eɪ",
	"ould": "ʊd",
	// Vowels (simplified)
	"ee": "iː",
	"ea": "iː",
	"oo": "uː",
	"ou": "aʊ",
	"oi": "ɔɪ",
	"oy": "ɔɪ",
	"ai": "eɪ",
	"ay": "eɪ",
	"aw": "ɔː",
	"au": "ɔː",
	"ew": "juː",
	// Single letters
	"a": "æ",
	"b": "b",
	"c": "k",
	"d": "d",
	"e": "ɛ",
	"f": "f",
	"g": "g",
	"h": "h",
	"i": "ɪ",
	"j": "dʒ",
	"k": "k",
	"l": "l",
	"m": "m",
	"n": "n",
	"o": "ɒ",
	"p": "p",
	"q": "k",
	"r": "r",
	"s": "s",
	"t": "t",
	"u": "ʌ",
	"v": "v",
	"w": "w",
	"x": "ks",
	"y": "j",
	"z": "z",
}

// Turkish phonetic rules
var turkishRules = map[string]string{
	// Turkish-specific characters
	"ç": "tʃ",
	"ş": "ʃ",
	"ğ": "ː", // lengthens preceding vowel
	"ı": "ɯ",
	"ö": "ø",
	"ü": "y",
	// Standard letters (Turkish is largely phonetic)
	"a": "a",
	"b": "b",
	"c": "dʒ",
	"d": "d",
	"e": "e",
	"f": "f",
	"g": "g",
	"h": "h",
	"i": "i",
	"j": "ʒ",
	"k": "k",
	"l": "l",
	"m": "m",
	"n": "n",
	"o": "o",
	"p": "p",
	"r": "r",
	"s": "s",
	"t": "t",
	"u": "u",
	"v": "v",
	"y": "j",
	"z": "z",
}

// German phonetic rules
var germanRules = map[string]string{
	// German-specific patterns
	"sch": "ʃ",
	"ch":  "x",
	"tsch": "tʃ",
	"tz":  "ts",
	"ß":   "s",
	"ä":   "ɛ",
	"ö":   "ø",
	"ü":   "y",
	"ie":  "iː",
	"ei":  "aɪ",
	"eu":  "ɔʏ",
	"äu":  "ɔʏ",
	"au":  "aʊ",
	// Standard letters
	"a": "a",
	"b": "b",
	"c": "k",
	"d": "d",
	"e": "e",
	"f": "f",
	"g": "g",
	"h": "h",
	"i": "i",
	"j": "j",
	"k": "k",
	"l": "l",
	"m": "m",
	"n": "n",
	"o": "o",
	"p": "p",
	"q": "k",
	"r": "r",
	"s": "s",
	"t": "t",
	"u": "u",
	"v": "f",
	"w": "v",
	"x": "ks",
	"y": "y",
	"z": "ts",
}

// French phonetic rules
var frenchRules = map[string]string{
	// French-specific patterns
	"ch":  "ʃ",
	"gn":  "ɲ",
	"qu":  "k",
	"ou":  "u",
	"oi":  "wa",
	"ai":  "ɛ",
	"ei":  "ɛ",
	"au":  "o",
	"eau": "o",
	"eu":  "ø",
	"œ":   "ø",
	"œu":  "ø",
	"an":  "ɑ̃",
	"en":  "ɑ̃",
	"in":  "ɛ̃",
	"on":  "ɔ̃",
	"un":  "œ̃",
	"é":   "e",
	"è":   "ɛ",
	"ê":   "ɛ",
	"ë":   "ɛ",
	"à":   "a",
	"â":   "ɑ",
	"î":   "i",
	"ï":   "i",
	"ô":   "o",
	"û":   "y",
	"ù":   "y",
	"ç":   "s",
	// Standard letters
	"a": "a",
	"b": "b",
	"c": "k",
	"d": "d",
	"e": "ə",
	"f": "f",
	"g": "g",
	"h": "", // silent in French
	"i": "i",
	"j": "ʒ",
	"k": "k",
	"l": "l",
	"m": "m",
	"n": "n",
	"o": "o",
	"p": "p",
	"q": "k",
	"r": "ʁ",
	"s": "s",
	"t": "t",
	"u": "y",
	"v": "v",
	"w": "w",
	"x": "ks",
	"y": "i",
	"z": "z",
}
