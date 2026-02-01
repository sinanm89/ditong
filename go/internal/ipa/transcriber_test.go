package ipa

import (
	"testing"
)

func TestNewTranscriber(t *testing.T) {
	tr := NewTranscriber("en")
	if tr.Language() != "en" {
		t.Errorf("Expected language 'en', got %q", tr.Language())
	}
}

func TestEnglishTranscription(t *testing.T) {
	tr := NewTranscriber("en")

	tests := []struct {
		word     string
		expected string
	}{
		{"the", "θɛ"},
		{"shop", "ʃɒp"},
		{"church", "tʃʌrtʃ"},
		{"phone", "fɒnɛ"},
		{"king", "kɪŋ"},
		{"hello", "hɛllɒ"},
		{"cat", "kæt"},
	}

	for _, tt := range tests {
		result := tr.Transcribe(tt.word)
		if result != tt.expected {
			t.Errorf("Transcribe(%q) = %q, want %q", tt.word, result, tt.expected)
		}
	}
}

func TestTurkishTranscription(t *testing.T) {
	tr := NewTranscriber("tr")

	tests := []struct {
		word     string
		expected string
	}{
		{"çay", "tʃaj"},
		{"şeker", "ʃeker"},
		{"görmek", "gørmek"},
		{"ışık", "ɯʃɯk"},
		{"merhaba", "merhaba"},
	}

	for _, tt := range tests {
		result := tr.Transcribe(tt.word)
		if result != tt.expected {
			t.Errorf("Transcribe(%q) = %q, want %q", tt.word, result, tt.expected)
		}
	}
}

func TestGermanTranscription(t *testing.T) {
	tr := NewTranscriber("de")

	tests := []struct {
		word     string
		expected string
	}{
		{"schön", "ʃøn"},
		{"ich", "ix"},
		{"zeit", "tsaɪt"},
		{"grüß", "grys"},  // ß -> s (single char)
		{"buch", "bux"},
	}

	for _, tt := range tests {
		result := tr.Transcribe(tt.word)
		if result != tt.expected {
			t.Errorf("Transcribe(%q) = %q, want %q", tt.word, result, tt.expected)
		}
	}
}

func TestFrenchTranscription(t *testing.T) {
	tr := NewTranscriber("fr")

	tests := []struct {
		word     string
		expected string
	}{
		{"chat", "ʃat"},     // t not silent in simplified rules
		{"bonjour", "bɔ̃ʒuʁ"},
		{"oiseau", "waso"},  // s -> s in simplified rules
		{"café", "kafe"},
	}

	for _, tt := range tests {
		result := tr.Transcribe(tt.word)
		if result != tt.expected {
			t.Errorf("Transcribe(%q) = %q, want %q", tt.word, result, tt.expected)
		}
	}
}

func TestUnknownLanguageFallback(t *testing.T) {
	tr := NewTranscriber("xyz")

	// Should use basic rules
	result := tr.Transcribe("cat")
	if result == "" {
		t.Error("Expected non-empty transcription for unknown language")
	}
}

func BenchmarkTranscribe(b *testing.B) {
	tr := NewTranscriber("en")
	word := "dictionary"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Transcribe(word)
	}
}
