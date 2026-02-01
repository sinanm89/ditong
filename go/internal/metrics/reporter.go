package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Reporter handles metrics output and history tracking.
type Reporter struct {
	outputDir   string
	historyFile string
}

// NewReporter creates a new metrics reporter.
func NewReporter(outputDir string) *Reporter {
	metricsDir := filepath.Join(outputDir, "metrics")
	os.MkdirAll(metricsDir, 0755)

	return &Reporter{
		outputDir:   metricsDir,
		historyFile: filepath.Join(metricsDir, "history.jsonl"),
	}
}

// Write writes run metrics to files.
func (r *Reporter) Write(metrics *RunMetrics) error {
	// Write latest.json (overwritten each run)
	latestPath := filepath.Join(r.outputDir, "latest.json")
	if err := r.writeJSON(latestPath, metrics); err != nil {
		return fmt.Errorf("failed to write latest.json: %w", err)
	}

	// Write timestamped file
	timestampedPath := filepath.Join(
		r.outputDir,
		fmt.Sprintf("run_%s.json", metrics.RunID),
	)
	if err := r.writeJSON(timestampedPath, metrics); err != nil {
		return fmt.Errorf("failed to write timestamped file: %w", err)
	}

	// Append to history
	if err := r.appendHistory(metrics); err != nil {
		return fmt.Errorf("failed to append history: %w", err)
	}

	return nil
}

// writeJSON writes a metrics struct to a JSON file.
func (r *Reporter) writeJSON(path string, metrics *RunMetrics) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(metrics)
}

// appendHistory appends a summary line to the history file.
func (r *Reporter) appendHistory(metrics *RunMetrics) error {
	file, err := os.OpenFile(r.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write compact JSON line
	line, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(line) + "\n")
	return err
}

// HistorySummary is a condensed view of a historical run.
type HistorySummary struct {
	RunID          string    `json:"run_id"`
	Timestamp      time.Time `json:"timestamp"`
	Languages      []string  `json:"languages"`
	TotalWords     int64     `json:"total_words"`
	DurationMs     int64     `json:"duration_ms"`
	Throughput     float64   `json:"throughput"`
}

// ReadHistory reads the last N runs from history.
func (r *Reporter) ReadHistory(limit int) ([]*RunMetrics, error) {
	file, err := os.Open(r.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var runs []*RunMetrics
	scanner := bufio.NewScanner(file)

	// Set a larger buffer for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		var run RunMetrics
		if err := json.Unmarshal(scanner.Bytes(), &run); err != nil {
			continue // Skip malformed lines
		}
		runs = append(runs, &run)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return only the last 'limit' runs
	if limit > 0 && len(runs) > limit {
		runs = runs[len(runs)-limit:]
	}

	return runs, nil
}

// GetLastRun returns the most recent run from history.
func (r *Reporter) GetLastRun() (*RunMetrics, error) {
	runs, err := r.ReadHistory(1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return runs[0], nil
}

// Compare generates a comparison between two runs.
type Comparison struct {
	CurrentRunID    string  `json:"current_run_id"`
	PreviousRunID   string  `json:"previous_run_id"`
	SpeedupFactor   float64 `json:"speedup_factor"`
	TimeSavedMs     int64   `json:"time_saved_ms"`
	WordsDiff       int64   `json:"words_diff"`
	ThroughputDiff  float64 `json:"throughput_diff"`
}

// CompareRuns compares two runs and returns the difference.
func CompareRuns(current, previous *RunMetrics) *Comparison {
	if current == nil || previous == nil {
		return nil
	}

	speedup := float64(1)
	if current.Totals.DurationMs > 0 {
		speedup = float64(previous.Totals.DurationMs) / float64(current.Totals.DurationMs)
	}

	return &Comparison{
		CurrentRunID:   current.RunID,
		PreviousRunID:  previous.RunID,
		SpeedupFactor:  speedup,
		TimeSavedMs:    previous.Totals.DurationMs - current.Totals.DurationMs,
		WordsDiff:      current.Totals.WordsProcessed - previous.Totals.WordsProcessed,
		ThroughputDiff: current.Totals.Throughput - previous.Totals.Throughput,
	}
}

// FormatComparison returns a human-readable comparison string.
func FormatComparison(c *Comparison) string {
	if c == nil {
		return "No previous run to compare"
	}

	direction := "faster"
	if c.SpeedupFactor < 1 {
		direction = "slower"
	}

	return fmt.Sprintf(
		"%.2fx %s than previous run (%+dms, %+.0f words/sec)",
		c.SpeedupFactor,
		direction,
		-c.TimeSavedMs, // Negative because saved = previous - current
		c.ThroughputDiff,
	)
}
