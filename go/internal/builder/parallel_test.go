package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ditong/internal/schema"
)

func TestDictionaryBuilderParallelBuild(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewDictionaryBuilder(tmpDir, 3, 10)

	// Add test words - use existing createTestWord(normalized, language, category)
	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("world", "en", "standard"),
		createTestWord("test", "en", "standard"),
		createTestWord("cat", "en", "standard"),
	}
	builder.AddWords(words, "en")

	words2 := []*schema.Word{
		createTestWord("hallo", "de", "standard"),
		createTestWord("welt", "de", "standard"),
	}
	builder.AddWords(words2, "de")

	config := ParallelBuildConfig{Workers: 4}
	stats := builder.ParallelBuild(context.Background(), config)

	if stats.TotalWords != 6 {
		t.Errorf("Expected 6 total words, got %d", stats.TotalWords)
	}

	if stats.ByLanguage["en"] != 4 {
		t.Errorf("Expected 4 English words, got %d", stats.ByLanguage["en"])
	}

	if stats.ByLanguage["de"] != 2 {
		t.Errorf("Expected 2 German words, got %d", stats.ByLanguage["de"])
	}

	// Verify files were created
	for _, f := range stats.FilesWritten {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("Expected file to exist: %s", f)
		}
	}
}

func TestDictionaryBuilderParallelBuildFallback(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewDictionaryBuilder(tmpDir, 3, 10)

	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
	}
	builder.AddWords(words, "en")

	// With workers=1, should fall back to sequential
	config := ParallelBuildConfig{Workers: 1}
	stats := builder.ParallelBuild(context.Background(), config)

	if stats.TotalWords != 1 {
		t.Errorf("Expected 1 word, got %d", stats.TotalWords)
	}
}

func TestSynthesisBuilderParallelBuild(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	// Add words from multiple languages
	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("world", "en", "standard"),
		createTestWord("hallo", "de", "standard"),
		createTestWord("apple", "en", "standard"),
		createTestWord("beach", "en", "standard"),
	}

	for _, w := range words {
		builder.AddWords([]*schema.Word{w})
	}

	synthConfig := NewSynthesisConfig("test_synth")
	synthConfig.IncludeLanguages["en"] = true
	synthConfig.IncludeLanguages["de"] = true
	synthConfig.IncludeCategories["standard"] = true
	synthConfig.MinLength = 3
	synthConfig.MaxLength = 10
	synthConfig.SplitByLetter = true

	parallelConfig := ParallelBuildConfig{Workers: 4}
	stats := builder.ParallelBuild(context.Background(), synthConfig, parallelConfig)

	if stats.TotalWords != 5 {
		t.Errorf("Expected 5 words, got %d", stats.TotalWords)
	}

	// Verify synthesis directory exists
	synthDir := filepath.Join(tmpDir, "synthesis", "test_synth")
	if _, err := os.Stat(synthDir); os.IsNotExist(err) {
		t.Errorf("Expected synthesis directory to exist: %s", synthDir)
	}
}

func TestSynthesisBuilderParallelBuildNoSplit(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewSynthesisBuilder(tmpDir)

	words := []*schema.Word{
		createTestWord("hello", "en", "standard"),
		createTestWord("world", "en", "standard"),
	}

	for _, w := range words {
		builder.AddWords([]*schema.Word{w})
	}

	synthConfig := NewSynthesisConfig("nosplit_synth")
	synthConfig.IncludeLanguages["en"] = true
	synthConfig.IncludeCategories["standard"] = true
	synthConfig.MinLength = 3
	synthConfig.MaxLength = 10
	synthConfig.SplitByLetter = false

	parallelConfig := ParallelBuildConfig{Workers: 4}
	stats := builder.ParallelBuild(context.Background(), synthConfig, parallelConfig)

	if stats.TotalWords != 2 {
		t.Errorf("Expected 2 words, got %d", stats.TotalWords)
	}

	// Should have a single file per length, not per letter
	found5c := false
	for _, f := range stats.FilesWritten {
		if filepath.Base(f) == "5-c.json" {
			found5c = true
			break
		}
	}
	if !found5c {
		t.Error("Expected 5-c.json file when not splitting by letter")
	}
}

func BenchmarkDictionaryBuild(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a builder with many words
	builder := NewDictionaryBuilder(tmpDir, 3, 10)
	for i := 0; i < 10000; i++ {
		word := schema.NewWord("word"+string(rune('a'+i%26)), 5, "5-c")
		word.AddSource(schema.WordSource{
			DictName: "test",
			Language: "en",
			Category: "standard",
		})
		builder.AddWords([]*schema.Word{word}, "en")
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder.Build()
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		config := ParallelBuildConfig{Workers: 4}
		for i := 0; i < b.N; i++ {
			builder.ParallelBuild(context.Background(), config)
		}
	})
}
