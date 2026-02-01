package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCursewordURLs(t *testing.T) {
	// Verify we have curseword URLs for major languages
	required := []string{"en", "tr", "de", "fr"}
	for _, lang := range required {
		if _, ok := CursewordURLs[lang]; !ok {
			t.Errorf("Missing curseword URL for language: %s", lang)
		}
	}
}

func TestCursewordConfig(t *testing.T) {
	cfg := CursewordConfig("en")

	if cfg.Language != "en" {
		t.Errorf("Expected language 'en', got %q", cfg.Language)
	}

	if cfg.Category != "curseword" {
		t.Errorf("Expected category 'curseword', got %q", cfg.Category)
	}
}

func TestIngestCursewords(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_cursewords.txt")

	// Create test curseword list
	content := `# Test curseword list
badword
another
test
short
ab
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := IngestConfig{
		Language:  "en",
		Category:  "curseword",
		MinLength: 3,
		MaxLength: 10,
	}

	result, err := IngestCursewords(testFile, config)
	if err != nil {
		t.Fatalf("IngestCursewords failed: %v", err)
	}

	// Should skip comment, skip "ab" (too short), keep 4 words
	if result.TotalValid != 4 {
		t.Errorf("Expected 4 valid words, got %d", result.TotalValid)
	}

	if result.Category != "curseword" {
		t.Errorf("Expected category 'curseword', got %q", result.Category)
	}

	// Verify curseword tag is set
	for _, word := range result.Words {
		if !word.Tags["curseword"] {
			t.Errorf("Word %q missing curseword tag", word.Normalized)
		}
	}
}

func TestHasCursewordSupport(t *testing.T) {
	if !HasCursewordSupport("en") {
		t.Error("Expected curseword support for 'en'")
	}

	if HasCursewordSupport("xyz") {
		t.Error("Expected no curseword support for 'xyz'")
	}
}

func TestGetCursewordLanguages(t *testing.T) {
	langs := GetCursewordLanguages()

	if len(langs) == 0 {
		t.Error("Expected at least one curseword language")
	}

	// Verify English is in the list
	found := false
	for _, lang := range langs {
		if lang == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'en' in curseword languages")
	}
}
