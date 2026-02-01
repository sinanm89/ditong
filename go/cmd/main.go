// ditong CLI - Multi-language lexicon toolkit.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"ditong/internal/builder"
	"ditong/internal/ingest"
	"ditong/internal/metrics"
	"ditong/internal/ui"

	"github.com/spf13/pflag"
)

func main() {
	// Flags
	languages := pflag.StringP("languages", "l", "en,tr", "Comma-separated language codes")
	minLength := pflag.Int("min-length", 3, "Minimum word length")
	maxLength := pflag.Int("max-length", 10, "Maximum word length")
	outputDir := pflag.StringP("output-dir", "o", "", "Output directory for dictionaries")
	cacheDir := pflag.StringP("cache-dir", "c", "", "Cache directory for downloaded sources")
	synthesis := pflag.StringP("synthesis", "s", "", "Name for synthesis dictionary")
	force := pflag.BoolP("force", "f", false, "Force re-download of dictionaries")
	noSplit := pflag.Bool("no-split", false, "Don't split synthesis by first letter")
	quiet := pflag.BoolP("quiet", "q", false, "Suppress progress output")
	verbose := pflag.BoolP("verbose", "v", false, "Verbose logging")
	writeMetrics := pflag.Bool("metrics", true, "Write metrics to output directory")
	benchmark := pflag.Bool("benchmark", false, "Run in benchmark mode (JSON output only)")

	// Parallel processing flags
	parallel := pflag.BoolP("parallel", "p", true, "Enable parallel processing")
	workers := pflag.IntP("workers", "w", 0, "Number of parallel workers (0 = auto)")

	pflag.Parse()

	// Auto-detect workers
	if *workers <= 0 {
		*workers = runtime.NumCPU()
		if *workers > 8 {
			*workers = 8 // Cap at 8 for network I/O
		}
	}

	// Initialize UI
	term := ui.New(*quiet || *benchmark, *verbose)

	// Print banner
	if !*benchmark {
		term.Banner()
	}

	// Get base directory
	exe, _ := os.Executable()
	baseDir := filepath.Dir(filepath.Dir(exe))
	if baseDir == "." {
		baseDir, _ = os.Getwd()
		baseDir = filepath.Dir(baseDir)
	}

	if *outputDir == "" {
		*outputDir = filepath.Join(baseDir, "dicts")
	}
	if *cacheDir == "" {
		*cacheDir = filepath.Join(baseDir, "sources")
	}

	// Parse languages
	langs := strings.Split(*languages, ",")
	for i := range langs {
		langs[i] = strings.TrimSpace(langs[i])
	}

	// Initialize metrics collector
	collector := metrics.NewCollector()
	collector.SetConfigMap(map[string]interface{}{
		"languages":  langs,
		"min_length": *minLength,
		"max_length": *maxLength,
		"parallel":   *parallel,
		"workers":    *workers,
	})

	// Show configuration
	if !*benchmark {
		term.ConfigWithParallel(langs, *minLength, *maxLength, *outputDir, *parallel, *workers)
	}

	// Initialize builders
	dictBuilder := builder.NewDictionaryBuilder(*outputDir, *minLength, *maxLength)
	synthBuilder := builder.NewSynthesisBuilder(*outputDir)

	// Phase 1: Download and ingest
	collector.StartStage("download")
	if !*benchmark {
		term.Phase(1, 3, "Downloading and ingesting dictionaries")
	}

	var totalRaw, totalValid int64

	if *parallel && len(langs) > 1 {
		// Parallel download and ingest
		if !*benchmark {
			term.Info(fmt.Sprintf("Parallel mode: %d workers", *workers))
		}

		parallelConfig := ingest.ParallelConfig{
			Workers:   *workers,
			Force:     *force,
			MinLength: *minLength,
			MaxLength: *maxLength,
		}

		// Track progress with mutex for thread-safe UI updates
		var mu sync.Mutex
		completedCount := 0

		results := ingest.ParallelDownloadAndIngest(langs, *cacheDir, parallelConfig, func(lang string, r *ingest.LanguageResult) {
			mu.Lock()
			defer mu.Unlock()
			completedCount++

			if *benchmark {
				return
			}

			if r.Error != nil {
				term.LanguageStatus(lang, "error", r.Error.Error())
			} else {
				status := "ok"
				details := fmt.Sprintf("%d words", r.Result.TotalValid)
				if r.Cached {
					details += " (cached)"
				}
				term.LanguageStatus(lang, status, details)
			}
		})

		// Aggregate results
		for _, r := range results {
			if r.Error != nil {
				continue
			}
			totalRaw += int64(r.Result.TotalRaw)
			totalValid += int64(r.Result.TotalValid)
			dictBuilder.AddWords(r.Result.Words, r.Language)
			synthBuilder.AddWords(r.Result.Words)
		}

		pstats := ingest.AggregateResults(results)
		collector.SetStageCounter("download", "cached", int64(pstats.Cached))
		collector.SetStageCounter("download", "failed", int64(pstats.Failed))

	} else {
		// Sequential download and ingest
		for _, lang := range langs {
			var spinner *ui.SpinnerWrapper
			if !*benchmark {
				spinner = term.Spinner(fmt.Sprintf("Processing %s...", strings.ToUpper(lang)))
			}

			config := ingest.DefaultConfig(lang)
			config.MinLength = *minLength
			config.MaxLength = *maxLength

			langCacheDir := filepath.Join(*cacheDir, lang)
			result, err := ingest.DownloadAndIngest(lang, langCacheDir, config, *force)

			if spinner != nil {
				spinner.Stop()
			}

			if err != nil {
				if !*benchmark {
					term.LanguageStatus(lang, "error", err.Error())
				}
				continue
			}

			totalRaw += int64(result.TotalRaw)
			totalValid += int64(result.TotalValid)

			if !*benchmark {
				term.LanguageStatus(lang, "ok", fmt.Sprintf("%d words ingested", result.TotalValid))
			}
			dictBuilder.AddWords(result.Words, lang)
			synthBuilder.AddWords(result.Words)
		}
	}

	collector.EndStage("download")
	collector.SetStageCounter("download", "words_raw", totalRaw)
	collector.SetStageCounter("download", "words_valid", totalValid)
	collector.SetStageCounter("download", "languages", int64(len(langs)))

	// Phase 2: Build per-language dictionaries
	collector.StartStage("build")
	if !*benchmark {
		term.Phase(2, 3, "Building per-language dictionaries")
	}

	var buildSpinner *ui.SpinnerWrapper
	if !*benchmark {
		buildSpinner = term.Spinner("Writing dictionary files...")
	}
	stats := dictBuilder.Build()
	if buildSpinner != nil {
		buildSpinner.Stop()
	}
	collector.EndStage("build")
	collector.SetStageCounter("build", "words", int64(stats.TotalWords))
	collector.SetStageCounter("build", "files", int64(len(stats.FilesWritten)))

	if !*benchmark {
		term.LanguageStats(stats.ByLanguage)
		term.Info(fmt.Sprintf("Total: %d words in %d files", stats.TotalWords, len(stats.FilesWritten)))
	}

	// Phase 3: Build synthesis dictionary
	collector.StartStage("synthesis")
	if !*benchmark {
		term.Phase(3, 3, "Building synthesis dictionary")
	}

	synthName := *synthesis
	if synthName == "" {
		sortedLangs := make([]string, len(langs))
		copy(sortedLangs, langs)
		sort.Strings(sortedLangs)
		synthName = strings.Join(sortedLangs, "_") + "_standard"
	}

	synthConfig := builder.NewSynthesisConfig(synthName)
	for _, lang := range langs {
		synthConfig.IncludeLanguages[lang] = true
	}
	synthConfig.IncludeCategories["standard"] = true
	synthConfig.MinLength = *minLength
	synthConfig.MaxLength = *maxLength
	synthConfig.SplitByLetter = !*noSplit

	var synthSpinner *ui.SpinnerWrapper
	if !*benchmark {
		synthSpinner = term.Spinner(fmt.Sprintf("Building synthesis: %s...", synthName))
	}
	synthStats := synthBuilder.Build(synthConfig)
	if synthSpinner != nil {
		synthSpinner.Stop()
	}
	collector.EndStage("synthesis")
	collector.SetStageCounter("synthesis", "unique_words", int64(synthStats.TotalWords))
	collector.SetStageCounter("synthesis", "files", int64(len(synthStats.FilesWritten)))

	if !*benchmark {
		term.Info(fmt.Sprintf("Synthesis: %s", synthName))
		term.LengthStats(synthStats.ByLength)
		term.Info(fmt.Sprintf("Unique words: %d in %d files", synthStats.TotalWords, len(synthStats.FilesWritten)))
	}

	// Finalize metrics
	totalFiles := len(stats.FilesWritten) + len(synthStats.FilesWritten)
	runMetrics := collector.Finalize(int64(stats.TotalWords), totalFiles)

	// Write metrics
	if *writeMetrics || *benchmark {
		reporter := metrics.NewReporter(*outputDir)

		// Get previous run for comparison
		previousRun, _ := reporter.GetLastRun()

		if err := reporter.Write(runMetrics); err != nil {
			if !*benchmark {
				term.Warning(fmt.Sprintf("Failed to write metrics: %v", err))
			}
		} else if !*benchmark {
			term.Debug(fmt.Sprintf("Metrics written: %s", runMetrics.RunID))
		}

		// Show comparison if available
		if previousRun != nil && !*benchmark {
			comparison := metrics.CompareRuns(runMetrics, previousRun)
			if comparison != nil {
				term.Info(metrics.FormatComparison(comparison))
			}
		}
	}

	// Final report
	if *benchmark {
		// In benchmark mode, output JSON metrics
		fmt.Printf(`{"run_id":"%s","duration_ms":%d,"throughput":%.2f,"words":%d,"files":%d,"parallel":%t,"workers":%d}`,
			runMetrics.RunID,
			runMetrics.Totals.DurationMs,
			runMetrics.Totals.Throughput,
			runMetrics.Totals.WordsProcessed,
			runMetrics.Totals.FilesWritten,
			*parallel,
			*workers,
		)
		fmt.Println()
	} else {
		term.FinalReport(stats.TotalWords, totalFiles, collector.GetStageDuration("download")+collector.GetStageDuration("build")+collector.GetStageDuration("synthesis"))
		term.Done()
	}
}
