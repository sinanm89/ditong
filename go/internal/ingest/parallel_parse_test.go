package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParallelIngestHunspell(t *testing.T) {
	// Create temp file with test data
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.dic")

	// Generate test content with enough lines to trigger parallel processing
	var content string
	content = "5000\n" // Word count header
	words := []string{"hello", "world", "testing", "sample", "alpha", "beta", "gamma", "delta"}
	for i := 0; i < 5000; i++ {
		content += words[i%len(words)] + "\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := IngestConfig{
		Language:  "en",
		Category:  "standard",
		MinLength: 3,
		MaxLength: 10,
	}

	parseConfig := ParseConfig{
		Workers:   4,
		ChunkSize: 500,
	}

	result, err := ParallelIngestHunspell(testFile, config, parseConfig)
	if err != nil {
		t.Fatalf("ParallelIngestHunspell failed: %v", err)
	}

	// Should have 8 unique words
	if result.TotalValid != 8 {
		t.Errorf("Expected 8 unique words, got %d", result.TotalValid)
	}

	// Should have processed 5000 raw lines
	if result.TotalRaw != 5000 {
		t.Errorf("Expected 5000 raw lines, got %d", result.TotalRaw)
	}

	// Verify language is set
	if result.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", result.Language)
	}
}

func TestParallelIngestFallsBackForSmallFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "small.dic")

	content := "5\nhello\nworld\ntest\nalpha\nbeta\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := IngestConfig{
		Language:  "en",
		Category:  "standard",
		MinLength: 3,
		MaxLength: 10,
	}

	parseConfig := ParseConfig{
		Workers:   4,
		ChunkSize: 1000,
	}

	result, err := ParallelIngestHunspell(testFile, config, parseConfig)
	if err != nil {
		t.Fatalf("ParallelIngestHunspell failed: %v", err)
	}

	if result.TotalValid != 5 {
		t.Errorf("Expected 5 unique words, got %d", result.TotalValid)
	}
}

func BenchmarkParallelIngest(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.dic")

	// Create a larger file for benchmarking
	var content string
	content = "50000\n"
	words := []string{"hello", "world", "testing", "sample", "alpha", "beta", "gamma", "delta",
		"epsilon", "zeta", "theta", "iota", "kappa", "lambda", "omega"}
	for i := 0; i < 50000; i++ {
		content += words[i%len(words)] + "\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	config := IngestConfig{
		Language:  "en",
		Category:  "standard",
		MinLength: 3,
		MaxLength: 10,
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			IngestHunspell(testFile, config)
		}
	})

	b.Run("Parallel-4workers", func(b *testing.B) {
		parseConfig := ParseConfig{Workers: 4, ChunkSize: 5000}
		for i := 0; i < b.N; i++ {
			ParallelIngestHunspell(testFile, config, parseConfig)
		}
	})

	b.Run("Parallel-8workers", func(b *testing.B) {
		parseConfig := ParseConfig{Workers: 8, ChunkSize: 5000}
		for i := 0; i < b.N; i++ {
			ParallelIngestHunspell(testFile, config, parseConfig)
		}
	})
}
