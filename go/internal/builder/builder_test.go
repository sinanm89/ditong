package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ditong/internal/schema"
)

func createTestWord(normalized, language, category string) *schema.Word {
	word := schema.NewWord(normalized, len(normalized), schema.WordTypeFromLength(len(normalized)))
	word.AddSource(schema.WordSource{
		DictName:     "test_" + language,
		DictFilepath: "/test/" + language + ".dic",
		Language:     language,
		OriginalForm: normalized,
		Category:     category,
	})
	return word
}

func TestNewDictionaryBuilder(t *testing.T) {
	builder := NewDictionaryBuilder("/tmp/test", 5, 8)

	if builder.OutputDir != "/tmp/test" {
		t.Errorf("OutputDir = %q, want /tmp/test", builder.OutputDir)
	}
	if builder.MinLength != 5 {
		t.Errorf("MinLength = %d, want 5", builder.MinLength)
	}
	if builder.MaxLength != 8 {
		t.Errorf("MaxLength = %d, want 8", builder.MaxLength)
	}
}

func TestDictionaryBuilderAddWords(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewDictionaryBuilder(tmpDir, 3, 10)

	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("world", "en", "standard"),
	}

	builder.AddWords(words, "en")

	// Verify internal state
	if _, ok := builder.words["en"]; !ok {
		t.Error("Builder should have 'en' language")
	}
	if len(builder.words["en"][5]) != 2 {
		t.Errorf("Builder should have 2 words of length 5, got %d", len(builder.words["en"][5]))
	}
}

func TestDictionaryBuilderBuild(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewDictionaryBuilder(tmpDir, 4, 6)

	builder.AddWords([]*schema.Word{
		createTestWord("test", "en", "standard"),   // 4 chars
		createTestWord("hello", "en", "standard"),  // 5 chars
		createTestWord("worlds", "en", "standard"), // 6 chars
	}, "en")

	stats := builder.Build()

	if stats.TotalWords != 3 {
		t.Errorf("TotalWords = %d, want 3", stats.TotalWords)
	}
	if len(stats.FilesWritten) != 3 {
		t.Errorf("FilesWritten = %d, want 3", len(stats.FilesWritten))
	}

	// Check files exist
	for _, expected := range []string{"4-c.json", "5-c.json", "6-c.json"} {
		path := filepath.Join(tmpDir, "en", expected)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", path)
		}
	}
}

func TestDictionaryBuilderBuildContent(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewDictionaryBuilder(tmpDir, 5, 5)
	builder.AddWords([]*schema.Word{createTestWord("hello", "en", "standard")}, "en")
	builder.Build()

	jsonPath := filepath.Join(tmpDir, "en", "5-c.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result["name"] != "en_5-c" {
		t.Errorf("name = %v, want en_5-c", result["name"])
	}
	if result["language"] != "en" {
		t.Errorf("language = %v, want en", result["language"])
	}

	words := result["words"].(map[string]interface{})
	if _, ok := words["hello"]; !ok {
		t.Error("words should contain 'hello'")
	}
}

func TestNewSynthesisConfig(t *testing.T) {
	config := NewSynthesisConfig("test_synth")

	if config.Name != "test_synth" {
		t.Errorf("Name = %q, want test_synth", config.Name)
	}
	if config.MinLength != 3 {
		t.Errorf("MinLength = %d, want 3", config.MinLength)
	}
	if config.MaxLength != 10 {
		t.Errorf("MaxLength = %d, want 10", config.MaxLength)
	}
	if !config.SplitByLetter {
		t.Error("SplitByLetter should be true by default")
	}
}

func TestNewSynthesisBuilder(t *testing.T) {
	builder := NewSynthesisBuilder("/tmp/test")

	expected := filepath.Join("/tmp/test", "synthesis")
	if builder.OutputDir != expected {
		t.Errorf("OutputDir = %q, want %q", builder.OutputDir, expected)
	}
}

func TestSynthesisBuilderAddWords(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("world", "en", "standard"),
	}
	builder.AddWords(words)

	if len(builder.wordPool) != 2 {
		t.Errorf("wordPool size = %d, want 2", len(builder.wordPool))
	}
}

func TestSynthesisBuilderAddWordsMerge(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	word1 := createTestWord("care", "en", "standard")

	word2 := schema.NewWord("care", 4, "4-c")
	word2.AddSource(schema.WordSource{
		DictName:     "test_tr",
		DictFilepath: "/tr.dic",
		Language:     "tr",
		OriginalForm: "çare",
		Category:     "standard",
	})

	builder.AddWords([]*schema.Word{word1})
	builder.AddWords([]*schema.Word{word2})

	if len(builder.wordPool) != 1 {
		t.Errorf("wordPool size = %d, want 1 (should merge)", len(builder.wordPool))
	}
	if len(builder.wordPool["care"].Sources) != 2 {
		t.Errorf("care sources = %d, want 2", len(builder.wordPool["care"].Sources))
	}
}

func TestSynthesisBuilderBuild(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	builder.AddWords([]*schema.Word{
		createTestWord("alpha", "en", "standard"),
		createTestWord("apple", "en", "standard"),
		createTestWord("beta", "tr", "standard"),
	})

	config := NewSynthesisConfig("test_synth")
	config.IncludeLanguages["en"] = true
	config.IncludeLanguages["tr"] = true
	config.IncludeCategories["standard"] = true
	config.MinLength = 4
	config.MaxLength = 6

	stats := builder.Build(config)

	if stats.TotalWords != 3 {
		t.Errorf("TotalWords = %d, want 3", stats.TotalWords)
	}
	if stats.ConfigName != "test_synth" {
		t.Errorf("ConfigName = %q, want test_synth", stats.ConfigName)
	}

	// Check config file created
	configFile := filepath.Join(tmpDir, "synthesis", "test_synth", "_config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file not created")
	}
}

func TestSynthesisBuilderBuildFilters(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	builder.AddWords([]*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("curse", "en", "curseword"),
		createTestWord("merhaba", "tr", "standard"),
	})

	config := NewSynthesisConfig("filtered")
	config.IncludeLanguages["en"] = true
	config.IncludeCategories["standard"] = true
	config.ExcludeCategories["curseword"] = true
	config.MinLength = 3
	config.MaxLength = 10

	stats := builder.Build(config)

	// Should only include "hello" (en, standard, not curseword)
	if stats.TotalWords != 1 {
		t.Errorf("TotalWords = %d, want 1", stats.TotalWords)
	}
	if !stats.LanguagesIncluded["en"] {
		t.Error("LanguagesIncluded should have 'en'")
	}
	if stats.LanguagesIncluded["tr"] {
		t.Error("LanguagesIncluded should not have 'tr'")
	}
}

func TestSynthesisBuilderBuildNoSplit(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	builder.AddWords([]*schema.Word{createTestWord("hello", "en", "standard")})

	config := NewSynthesisConfig("no_split")
	config.MinLength = 5
	config.MaxLength = 5
	config.SplitByLetter = false

	stats := builder.Build(config)

	// Should have single file per length
	expectedFile := filepath.Join(tmpDir, "synthesis", "no_split", "5-c.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file not created: %s", expectedFile)
	}

	// Should NOT have letter-split directory
	letterDir := filepath.Join(tmpDir, "synthesis", "no_split", "5-c", "h.json")
	if _, err := os.Stat(letterDir); !os.IsNotExist(err) {
		t.Error("Should not create letter-split files when SplitByLetter is false")
	}

	if len(stats.FilesWritten) != 2 { // config + 1 json
		t.Errorf("FilesWritten = %d, want 2", len(stats.FilesWritten))
	}
}

func TestBuildStats(t *testing.T) {
	stats := NewBuildStats()

	if stats.TotalWords != 0 {
		t.Errorf("TotalWords = %d, want 0", stats.TotalWords)
	}
	if len(stats.ByLength) != 0 {
		t.Errorf("ByLength should be empty")
	}
	if len(stats.FilesWritten) != 0 {
		t.Errorf("FilesWritten should be empty")
	}
}
