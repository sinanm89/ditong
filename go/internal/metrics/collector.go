// Package metrics provides performance metrics collection and reporting.
package metrics

import (
	"crypto/rand"
	"encoding/hex"
	"runtime"
	"time"
)

// StageMetrics holds metrics for a single processing stage.
type StageMetrics struct {
	Name       string        `json:"name"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	DurationMs int64         `json:"duration_ms"`
	Counters   map[string]int64   `json:"counters,omitempty"`
	Gauges     map[string]float64 `json:"gauges,omitempty"`
}

// RunMetrics holds all metrics for a complete run.
type RunMetrics struct {
	RunID       string                 `json:"run_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Config      map[string]interface{} `json:"config"`
	Stages      map[string]*StageMetrics `json:"stages"`
	Totals      *TotalMetrics          `json:"totals"`
	Environment *EnvironmentInfo       `json:"environment"`
}

// TotalMetrics holds aggregate metrics.
type TotalMetrics struct {
	DurationMs     int64   `json:"duration_ms"`
	PeakMemoryMB   float64 `json:"peak_memory_mb"`
	WordsProcessed int64   `json:"words_processed"`
	FilesWritten   int     `json:"files_written"`
	Throughput     float64 `json:"throughput_words_per_sec"`
}

// EnvironmentInfo holds system environment details.
type EnvironmentInfo struct {
	GoVersion   string `json:"go_version"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	NumCPU      int    `json:"num_cpu"`
	MaxProcs    int    `json:"max_procs"`
}

// Collector collects metrics during execution.
type Collector struct {
	runID       string
	startTime   time.Time
	config      map[string]interface{}
	stages      map[string]*StageMetrics
	activeStage string
	peakMemory  uint64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		runID:     generateRunID(),
		startTime: time.Now(),
		config:    make(map[string]interface{}),
		stages:    make(map[string]*StageMetrics),
	}
}

// generateRunID creates a unique run identifier.
func generateRunID() string {
	timestamp := time.Now().Format("20060102-150405")
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return timestamp + "-" + hex.EncodeToString(bytes)
}

// SetConfig stores configuration for the run.
func (c *Collector) SetConfig(key string, value interface{}) {
	c.config[key] = value
}

// SetConfigMap stores multiple configuration values.
func (c *Collector) SetConfigMap(config map[string]interface{}) {
	for k, v := range config {
		c.config[k] = v
	}
}

// StartStage begins timing a new processing stage.
func (c *Collector) StartStage(name string) {
	c.activeStage = name
	c.stages[name] = &StageMetrics{
		Name:      name,
		StartTime: time.Now(),
		Counters:  make(map[string]int64),
		Gauges:    make(map[string]float64),
	}
	c.updatePeakMemory()
}

// EndStage completes timing for the current stage.
func (c *Collector) EndStage(name string) {
	if stage, ok := c.stages[name]; ok {
		stage.EndTime = time.Now()
		stage.DurationMs = stage.EndTime.Sub(stage.StartTime).Milliseconds()
	}
	c.updatePeakMemory()
}

// IncrementCounter increments a counter for the active stage.
func (c *Collector) IncrementCounter(name string, delta int64) {
	if c.activeStage == "" {
		return
	}
	if stage, ok := c.stages[c.activeStage]; ok {
		stage.Counters[name] += delta
	}
}

// SetCounter sets a counter value for the active stage.
func (c *Collector) SetCounter(name string, value int64) {
	if c.activeStage == "" {
		return
	}
	if stage, ok := c.stages[c.activeStage]; ok {
		stage.Counters[name] = value
	}
}

// SetGauge sets a gauge value for the active stage.
func (c *Collector) SetGauge(name string, value float64) {
	if c.activeStage == "" {
		return
	}
	if stage, ok := c.stages[c.activeStage]; ok {
		stage.Gauges[name] = value
	}
}

// SetStageCounter sets a counter for a specific stage.
func (c *Collector) SetStageCounter(stage, name string, value int64) {
	if s, ok := c.stages[stage]; ok {
		s.Counters[name] = value
	}
}

// SetStageGauge sets a gauge for a specific stage.
func (c *Collector) SetStageGauge(stage, name string, value float64) {
	if s, ok := c.stages[stage]; ok {
		s.Gauges[name] = value
	}
}

// updatePeakMemory tracks the maximum memory usage.
func (c *Collector) updatePeakMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Alloc > c.peakMemory {
		c.peakMemory = m.Alloc
	}
}

// Finalize creates the final RunMetrics report.
func (c *Collector) Finalize(totalWords int64, filesWritten int) *RunMetrics {
	c.updatePeakMemory()
	totalDuration := time.Since(c.startTime)

	throughput := float64(0)
	if totalDuration.Seconds() > 0 {
		throughput = float64(totalWords) / totalDuration.Seconds()
	}

	return &RunMetrics{
		RunID:     c.runID,
		Timestamp: c.startTime,
		Config:    c.config,
		Stages:    c.stages,
		Totals: &TotalMetrics{
			DurationMs:     totalDuration.Milliseconds(),
			PeakMemoryMB:   float64(c.peakMemory) / 1024 / 1024,
			WordsProcessed: totalWords,
			FilesWritten:   filesWritten,
			Throughput:     throughput,
		},
		Environment: &EnvironmentInfo{
			GoVersion: runtime.Version(),
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
			NumCPU:    runtime.NumCPU(),
			MaxProcs:  runtime.GOMAXPROCS(0),
		},
	}
}

// GetRunID returns the run identifier.
func (c *Collector) GetRunID() string {
	return c.runID
}

// GetStageDuration returns the duration of a completed stage.
func (c *Collector) GetStageDuration(name string) time.Duration {
	if stage, ok := c.stages[name]; ok && !stage.EndTime.IsZero() {
		return stage.EndTime.Sub(stage.StartTime)
	}
	return 0
}
