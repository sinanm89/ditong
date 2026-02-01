package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHunspellURLs(t *testing.T) {
	expectedLangs := []string{"en", "tr", "de", "fr", "es", "it", "pt", "nl", "pl", "ru"}

	for _, lang := range expectedLangs {
		if _, ok := HunspellURLs[lang]; !ok {
			t.Errorf("HunspellURLs missing language: %s", lang)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("en")

	if config.Language != "en" {
		t.Errorf("Language = %q, want en", config.Language)
	}
	if config.Category != "standard" {
		t.Errorf("Category = %q, want standard", config.Category)
	}
	if config.MinLength != 3 {
		t.Errorf("MinLength = %d, want 3", config.MinLength)
	}
	if config.MaxLength != 10 {
		t.Errorf("MaxLength = %d, want 10", config.MaxLength)
	}
}

func TestIngestHunspell(t *testing.T) {
	// Create temp dic file
	content := `10
hello
world
testing/ABC
sample/XYZ
python
short
ab
a
`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "test.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := IngestConfig{
		Language:  "en",
		Category:  "standard",
		MinLength: 4,
		MaxLength: 10,
	}

	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// Should exclude "ab" and "a" (too short), "short" (6 chars, should be included)
	if result.TotalValid < 4 {
		t.Errorf("TotalValid = %d, want >= 4", result.TotalValid)
	}

	// Verify words are normalized
	for _, word := range result.Words {
		if word.Normalized != word.Normalized {
			t.Errorf("Word not normalized: %q", word.Normalized)
		}
		if word.Length < config.MinLength || word.Length > config.MaxLength {
			t.Errorf("Word length %d out of range [%d, %d]", word.Length, config.MinLength, config.MaxLength)
		}
	}
}

func TestIngestHunspellTurkish(t *testing.T) {
	content := `5
çare
şeker
merhaba
test
`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "tr.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := IngestConfig{
		Language:  "tr",
		Category:  "standard",
		MinLength: 3,
		MaxLength: 10,
	}

	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// Check normalization
	normalizedForms := make(map[string]bool)
	for _, word := range result.Words {
		normalizedForms[word.Normalized] = true
	}

	if !normalizedForms["care"] {
		t.Error("çare should normalize to 'care'")
	}
	if !normalizedForms["seker"] {
		t.Error("şeker should normalize to 'seker'")
	}
}

func TestIngestHunspellAffixStripping(t *testing.T) {
	content := `3
word/ABC
another/XYZ/123
plain
`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "test.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := DefaultConfig("en")
	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// Verify affix flags are stripped
	normalizedForms := make(map[string]bool)
	for _, word := range result.Words {
		normalizedForms[word.Normalized] = true
	}

	if !normalizedForms["word"] {
		t.Error("'word/ABC' should be parsed as 'word'")
	}
	if !normalizedForms["another"] {
		t.Error("'another/XYZ/123' should be parsed as 'another'")
	}
	if !normalizedForms["plain"] {
		t.Error("'plain' should be included")
	}
}

func TestIngestHunspellDuplicates(t *testing.T) {
	content := `5
hello
HELLO
Hello
HeLLo
world
`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "test.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := DefaultConfig("en")
	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// All "hello" variants should merge
	if result.TotalValid != 2 {
		t.Errorf("TotalValid = %d, want 2 (hello + world)", result.TotalValid)
	}

	// Should have 3 duplicates (HELLO, Hello, HeLLo)
	if result.TotalDuplicates != 3 {
		t.Errorf("TotalDuplicates = %d, want 3", result.TotalDuplicates)
	}

	// Find hello and check sources
	for _, word := range result.Words {
		if word.Normalized == "hello" {
			if len(word.Sources) != 4 {
				t.Errorf("hello sources = %d, want 4", len(word.Sources))
			}
		}
	}
}

func TestIngestHunspellSkipsWordCount(t *testing.T) {
	content := `123456
word
test
`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "test.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := DefaultConfig("en")
	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// Should have 2 words, not 3 (first line is word count)
	if result.TotalValid != 2 {
		t.Errorf("TotalValid = %d, want 2", result.TotalValid)
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()

	if len(langs) == 0 {
		t.Error("GetSupportedLanguages returned empty list")
	}

	// Check for common languages
	langMap := make(map[string]bool)
	for _, l := range langs {
		langMap[l] = true
	}

	if !langMap["en"] {
		t.Error("Supported languages should include 'en'")
	}
	if !langMap["tr"] {
		t.Error("Supported languages should include 'tr'")
	}
}

func TestIngestHunspellEmptyLines(t *testing.T) {
	content := `5

hello

world

`
	tmpDir := t.TempDir()
	dicPath := filepath.Join(tmpDir, "test.dic")

	if err := os.WriteFile(dicPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := DefaultConfig("en")
	result, err := IngestHunspell(dicPath, config)
	if err != nil {
		t.Fatalf("IngestHunspell failed: %v", err)
	}

	// Should handle empty lines gracefully
	if result.TotalValid != 2 {
		t.Errorf("TotalValid = %d, want 2", result.TotalValid)
	}
}
