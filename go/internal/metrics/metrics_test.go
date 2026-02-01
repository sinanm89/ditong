package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	c := NewCollector()

	// Test run ID generation
	if c.GetRunID() == "" {
		t.Error("Expected non-empty run ID")
	}

	// Test config
	c.SetConfig("languages", []string{"en", "tr"})
	c.SetConfig("parallel", true)

	// Test stage tracking
	c.StartStage("download")
	time.Sleep(10 * time.Millisecond)
	c.IncrementCounter("files", 2)
	c.SetGauge("bytes_per_sec", 1024.5)
	c.EndStage("download")

	c.StartStage("ingest")
	c.SetCounter("words_raw", 10000)
	c.SetCounter("words_valid", 8500)
	c.EndStage("ingest")

	// Test finalize
	metrics := c.Finalize(8500, 16)

	if metrics.RunID == "" {
		t.Error("Expected non-empty run ID in metrics")
	}

	if metrics.Totals.WordsProcessed != 8500 {
		t.Errorf("Expected 8500 words, got %d", metrics.Totals.WordsProcessed)
	}

	if metrics.Totals.FilesWritten != 16 {
		t.Errorf("Expected 16 files, got %d", metrics.Totals.FilesWritten)
	}

	if _, ok := metrics.Stages["download"]; !ok {
		t.Error("Expected download stage in metrics")
	}

	if _, ok := metrics.Stages["ingest"]; !ok {
		t.Error("Expected ingest stage in metrics")
	}

	downloadStage := metrics.Stages["download"]
	if downloadStage.Counters["files"] != 2 {
		t.Errorf("Expected files counter = 2, got %d", downloadStage.Counters["files"])
	}

	ingestStage := metrics.Stages["ingest"]
	if ingestStage.Counters["words_valid"] != 8500 {
		t.Errorf("Expected words_valid = 8500, got %d", ingestStage.Counters["words_valid"])
	}
}

func TestReporter(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "metrics-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	reporter := NewReporter(tmpDir)

	// Create test metrics
	c := NewCollector()
	c.SetConfig("languages", []string{"en"})
	c.StartStage("test")
	c.SetCounter("items", 100)
	c.EndStage("test")
	metrics := c.Finalize(100, 5)

	// Test write
	if err := reporter.Write(metrics); err != nil {
		t.Fatalf("Failed to write metrics: %v", err)
	}

	// Verify files exist
	latestPath := filepath.Join(tmpDir, "metrics", "latest.json")
	if _, err := os.Stat(latestPath); os.IsNotExist(err) {
		t.Error("Expected latest.json to exist")
	}

	historyPath := filepath.Join(tmpDir, "metrics", "history.jsonl")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("Expected history.jsonl to exist")
	}

	// Test read history
	runs, err := reporter.ReadHistory(10)
	if err != nil {
		t.Fatalf("Failed to read history: %v", err)
	}

	if len(runs) != 1 {
		t.Errorf("Expected 1 run in history, got %d", len(runs))
	}

	// Test last run
	lastRun, err := reporter.GetLastRun()
	if err != nil {
		t.Fatalf("Failed to get last run: %v", err)
	}

	if lastRun.RunID != metrics.RunID {
		t.Errorf("Expected run ID %s, got %s", metrics.RunID, lastRun.RunID)
	}
}

func TestComparison(t *testing.T) {
	// Create two runs
	c1 := NewCollector()
	c1.SetConfig("test", true)
	metrics1 := c1.Finalize(1000, 10)
	metrics1.Totals.DurationMs = 1000
	metrics1.Totals.Throughput = 1000

	c2 := NewCollector()
	c2.SetConfig("test", true)
	metrics2 := c2.Finalize(1000, 10)
	metrics2.Totals.DurationMs = 500
	metrics2.Totals.Throughput = 2000

	// Compare
	comparison := CompareRuns(metrics2, metrics1)

	if comparison == nil {
		t.Fatal("Expected non-nil comparison")
	}

	if comparison.SpeedupFactor != 2.0 {
		t.Errorf("Expected 2x speedup, got %.2f", comparison.SpeedupFactor)
	}

	if comparison.TimeSavedMs != 500 {
		t.Errorf("Expected 500ms saved, got %d", comparison.TimeSavedMs)
	}

	// Test format
	formatted := FormatComparison(comparison)
	if formatted == "" {
		t.Error("Expected non-empty formatted comparison")
	}
}
