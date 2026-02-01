package normalizer

import (
	"testing"
)

func TestNormalizeChar(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected string
	}{
		// Turkish
		{"turkish c cedilla", 'ç', "c"},
		{"turkish C cedilla", 'Ç', "c"},
		{"turkish s cedilla", 'ş', "s"},
		{"turkish S cedilla", 'Ş', "s"},
		{"turkish g breve", 'ğ', "g"},
		{"turkish G breve", 'Ğ', "g"},
		{"turkish dotless i", 'ı', "i"},
		{"turkish dotted I", 'İ', "i"},
		{"turkish o umlaut", 'ö', "o"},
		{"turkish u umlaut", 'ü', "u"},

		// German
		{"german a umlaut", 'ä', "a"},
		{"german o umlaut", 'ö', "o"},
		{"german u umlaut", 'ü', "u"},
		{"german eszett", 'ß', "ss"},

		// French
		{"french e acute", 'é', "e"},
		{"french e grave", 'è', "e"},
		{"french a grave", 'à', "a"},

		// Spanish
		{"spanish n tilde", 'ñ', "n"},
		{"spanish a acute", 'á', "a"},

		// ASCII passthrough
		{"ascii lowercase", 'a', "a"},
		{"ascii uppercase", 'A', "a"},
		{"ascii z", 'z', "z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeChar(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeChar(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple lowercase", "hello", "hello"},
		{"simple uppercase", "HELLO", "hello"},
		{"mixed case", "Hello", "hello"},
		{"turkish care", "çare", "care"},
		{"turkish seker", "şeker", "seker"},
		{"turkish gormek", "görmek", "gormek"},
		{"turkish isik", "ışık", "isik"},
		{"german grosse", "größe", "grosse"},
		{"german uber", "über", "uber"},
		{"turkish uppercase", "ÇARE", "care"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeWord(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeWord(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWordEquivalence(t *testing.T) {
	// Test that care (EN) and çare (TR) normalize to the same value
	enWord := NormalizeWord("care")
	trWord := NormalizeWord("çare")

	if enWord != trWord {
		t.Errorf("care (%q) and çare (%q) should normalize to same value", enWord, trWord)
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "hello", true},
		{"valid single char", "a", true},
		{"valid long", "abcdefghijklmnopqrstuvwxyz", true},

		{"invalid uppercase", "Hello", false},
		{"invalid all caps", "HELLO", false},
		{"invalid with numbers", "hello123", false},
		{"invalid with hyphen", "hello-world", false},
		{"invalid with underscore", "hello_world", false},
		{"invalid with space", "hello world", false},
		{"invalid empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeAndValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid simple", "hello", "hello"},
		{"valid uppercase", "HELLO", "hello"},
		{"valid turkish", "çare", "care"},
		{"invalid with numbers", "hello123", ""},
		{"invalid with special", "hello!", ""},
		{"invalid empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeAndValidate(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeAndValidate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkNormalizeWord(b *testing.B) {
	words := []string{"hello", "çare", "größe", "merhaba", "testing"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, word := range words {
			NormalizeWord(word)
		}
	}
}

func BenchmarkIsValidIdentifier(b *testing.B) {
	words := []string{"hello", "world", "test123", "valid", "Hello"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, word := range words {
			IsValidIdentifier(word)
		}
	}
}
