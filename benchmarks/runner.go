// Benchmark runner for ditong.
// Run with: go run runner.go [options]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Parallel bool   `json:"parallel"`
	Workers  int    `json:"workers"`
}

type Group struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
	Configs     []Config `json:"configs"`
}

type ConfigFile struct {
	Groups []Group `json:"groups"`
}

type BenchmarkResult struct {
	ConfigID    string  `json:"config_id"`
	Group       string  `json:"group"`
	Languages   string  `json:"languages"`
	DurationMs  int64   `json:"duration_ms"`
	Throughput  float64 `json:"throughput"`
	Words       int     `json:"words"`
	Files       int     `json:"files"`
	Parallel    bool    `json:"parallel"`
	Workers     int     `json:"workers"`
}

func main() {
	configPath := flag.String("config", "configs.json", "Path to benchmark configs")
	outputDir := flag.String("output", "results", "Output directory for results")
	group := flag.String("group", "", "Run only this group (empty = all)")
	iterations := flag.Int("iterations", 1, "Number of iterations per config")
	force := flag.Bool("force", false, "Force re-download dictionaries")
	flag.Parse()

	// Load configs
	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	var cfg ConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Find ditong binary
	ditongPath := findDitong()
	if ditongPath == "" {
		fmt.Fprintln(os.Stderr, "Error: ditong binary not found. Build with 'go build' first.")
		os.Exit(1)
	}

	// Create output directory
	os.MkdirAll(*outputDir, 0755)

	// Run benchmarks
	var results []BenchmarkResult
	total := countConfigs(cfg.Groups, *group)
	current := 0

	for _, g := range cfg.Groups {
		if *group != "" && g.Name != *group {
			continue
		}

		fmt.Printf("\n=== Group: %s (%s) ===\n", g.Name, g.Description)
		fmt.Printf("Languages: %s\n", strings.Join(g.Languages, ", "))

		for _, c := range g.Configs {
			current++
			fmt.Printf("\n[%d/%d] Running: %s\n", current, total, c.Name)

			var durations []int64
			var lastResult BenchmarkResult

			for i := 0; i < *iterations; i++ {
				if *iterations > 1 {
					fmt.Printf("  Iteration %d/%d...", i+1, *iterations)
				}

				result, err := runBenchmark(ditongPath, g, c, *force && i == 0)
				if err != nil {
					fmt.Printf(" ERROR: %v\n", err)
					continue
				}

				durations = append(durations, result.DurationMs)
				lastResult = result

				if *iterations > 1 {
					fmt.Printf(" %dms\n", result.DurationMs)
				} else {
					fmt.Printf("  Duration: %dms, Words: %d, Files: %d\n",
						result.DurationMs, result.Words, result.Files)
				}
			}

			if len(durations) > 0 {
				// Use average for multiple iterations
				if *iterations > 1 {
					var sum int64
					for _, d := range durations {
						sum += d
					}
					lastResult.DurationMs = sum / int64(len(durations))
					fmt.Printf("  Average: %dms\n", lastResult.DurationMs)
				}
				results = append(results, lastResult)
			}
		}
	}

	// Write results
	resultsFile := filepath.Join(*outputDir, fmt.Sprintf("benchmark_%s.json",
		time.Now().Format("2006-01-02_15-04-05")))

	output := map[string]interface{}{
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"iterations": *iterations,
		"results":    results,
	}

	data, _ = json.MarshalIndent(output, "", "  ")
	if err := os.WriteFile(resultsFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing results: %v\n", err)
	} else {
		fmt.Printf("\nResults written to: %s\n", resultsFile)
	}

	// Print summary table
	printSummary(results)
}

func findDitong() string {
	// Check common locations
	candidates := []string{
		"../go/ditong",
		"../go/ditong.exe",
		"ditong",
		"ditong.exe",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Try to find in PATH
	path, err := exec.LookPath("ditong")
	if err == nil {
		return path
	}

	return ""
}

func countConfigs(groups []Group, filter string) int {
	count := 0
	for _, g := range groups {
		if filter != "" && g.Name != filter {
			continue
		}
		count += len(g.Configs)
	}
	return count
}

func runBenchmark(ditongPath string, g Group, c Config, force bool) (BenchmarkResult, error) {
	args := []string{
		"--benchmark",
		"--languages", strings.Join(g.Languages, ","),
		"--workers", fmt.Sprintf("%d", c.Workers),
	}

	if c.Parallel {
		args = append(args, "--parallel")
	} else {
		args = append(args, "--parallel=false")
	}

	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command(ditongPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("command failed: %w", err)
	}

	// Parse JSON output
	var result struct {
		RunID      string  `json:"run_id"`
		DurationMs int64   `json:"duration_ms"`
		Throughput float64 `json:"throughput"`
		Words      int     `json:"words"`
		Files      int     `json:"files"`
		Parallel   bool    `json:"parallel"`
		Workers    int     `json:"workers"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return BenchmarkResult{}, fmt.Errorf("failed to parse output: %w (output: %s)", err, string(output))
	}

	return BenchmarkResult{
		ConfigID:   c.ID,
		Group:      g.Name,
		Languages:  strings.Join(g.Languages, ","),
		DurationMs: result.DurationMs,
		Throughput: result.Throughput,
		Words:      result.Words,
		Files:      result.Files,
		Parallel:   result.Parallel,
		Workers:    result.Workers,
	}, nil
}

func printSummary(results []BenchmarkResult) {
	if len(results) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-30s %10s %10s %8s\n", "Config", "Duration", "Words", "Speedup")
	fmt.Println(strings.Repeat("-", 70))

	// Group results by group name
	groups := make(map[string][]BenchmarkResult)
	for _, r := range results {
		groups[r.Group] = append(groups[r.Group], r)
	}

	for groupName, groupResults := range groups {
		fmt.Printf("\n[%s]\n", groupName)

		// Find baseline (sequential)
		var baseline int64
		for _, r := range groupResults {
			if !r.Parallel {
				baseline = r.DurationMs
				break
			}
		}

		for _, r := range groupResults {
			speedup := "-"
			if baseline > 0 && r.DurationMs > 0 {
				speedupVal := float64(baseline) / float64(r.DurationMs)
				speedup = fmt.Sprintf("%.2fx", speedupVal)
			}

			name := r.ConfigID
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			fmt.Printf("%-30s %8dms %10d %8s\n",
				name, r.DurationMs, r.Words, speedup)
		}
	}

	fmt.Println(strings.Repeat("=", 70))
}
