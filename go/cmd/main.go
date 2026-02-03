// ditong CLI - Multi-language lexicon toolkit.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"ditong/internal/builder"
	"ditong/internal/config"
	"ditong/internal/ingest"
	"ditong/internal/ipa"
	"ditong/internal/metrics"
	"ditong/internal/ui"

	"github.com/pterm/pterm"
	"github.com/spf13/pflag"
)

// interactiveDefaults holds current flag values to use as defaults
type interactiveDefaults struct {
	languages   string
	minLength   int
	maxLength   int
	outputDir   string
	ipa         bool
	cursewords  bool
	consolidate bool
	force       bool
}

// runInteractiveMode prompts for all options interactively using pterm
// It skips prompts for flags that were explicitly set on command line
func runInteractiveMode(defaults interactiveDefaults) (langs string, minLen, maxLen int, outDir string, ipaEnabled, cursewords, consolidate, forceDownload bool) {
	// Print header
	fmt.Println()
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("di", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle("tong", pterm.NewStyle(pterm.FgLightBlue)),
	).Render()

	headerBox := pterm.DefaultBox.WithTitle(pterm.FgCyan.Sprint("Interactive Setup")).
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightBlue))

	headerBox.Println(
		pterm.FgWhite.Sprint("Made by ") + pterm.FgCyan.Sprint("Sinan Midillili") + pterm.FgWhite.Sprint(" - ") + pterm.FgLightBlue.Sprint("https://rahatol.com") + "\n" +
			pterm.FgCyan.Sprint("ditong") + pterm.FgGray.Sprint(" - ") + pterm.FgLightBlue.Sprint("https://github.com/sinanm89/ditong"),
	)
	fmt.Println()

	// Available languages info
	pterm.Info.Println("Available languages: " + pterm.FgCyan.Sprint(config.AvailableLanguagesStr()))
	fmt.Println()

	// Interactive prompts - skip if flag was explicitly set
	if isFlagSet("languages") {
		langs = defaults.languages
		pterm.Info.Println("Languages: " + pterm.FgCyan.Sprint(langs) + pterm.FgGray.Sprint(" (from --languages flag)"))
	} else {
		langs, _ = pterm.DefaultInteractiveTextInput.
			WithDefaultValue(defaults.languages).
			WithMultiLine(false).
			Show("Languages (comma-separated)")
	}

	if isFlagSet("min-length") {
		minLen = defaults.minLength
		pterm.Info.Println("Min length: " + pterm.FgCyan.Sprintf("%d", minLen) + pterm.FgGray.Sprint(" (from --min-length flag)"))
	} else {
		minLenStr, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultValue(strconv.Itoa(defaults.minLength)).
			Show("Minimum word length")
		minLen, _ = strconv.Atoi(minLenStr)
		if minLen < 1 {
			minLen = config.DefaultMinLength()
		}
	}

	if isFlagSet("max-length") {
		maxLen = defaults.maxLength
		pterm.Info.Println("Max length: " + pterm.FgCyan.Sprintf("%d", maxLen) + pterm.FgGray.Sprint(" (from --max-length flag)"))
	} else {
		maxLenStr, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultValue(strconv.Itoa(defaults.maxLength)).
			Show("Maximum word length")
		maxLen, _ = strconv.Atoi(maxLenStr)
		if maxLen < minLen {
			maxLen = config.DefaultMaxLength()
		}
	}

	if isFlagSet("output-dir") {
		outDir = defaults.outputDir
		pterm.Info.Println("Output dir: " + pterm.FgCyan.Sprint(outDir) + pterm.FgGray.Sprint(" (from --output-dir flag)"))
	} else {
		defaultOut := defaults.outputDir
		if defaultOut == "" {
			defaultOut = config.DefaultOutputDir()
		}
		outDir, _ = pterm.DefaultInteractiveTextInput.
			WithDefaultValue(defaultOut).
			Show("Output directory")
	}

	if isFlagSet("ipa") {
		ipaEnabled = defaults.ipa
		status := pterm.FgGray.Sprint("✗ Disabled")
		if ipaEnabled {
			status = pterm.FgGreen.Sprint("✓ Enabled")
		}
		pterm.Info.Println("IPA transcriptions: " + status + pterm.FgGray.Sprint(" (from --ipa flag)"))
	} else {
		ipaEnabled, _ = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(defaults.ipa).
			Show("Include IPA transcriptions?")
	}

	if isFlagSet("cursewords") {
		cursewords = defaults.cursewords
		status := pterm.FgGray.Sprint("✗ Disabled")
		if cursewords {
			status = pterm.FgGreen.Sprint("✓ Enabled")
		}
		pterm.Info.Println("Curseword dictionaries: " + status + pterm.FgGray.Sprint(" (from --cursewords flag)"))
	} else {
		cursewords, _ = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(defaults.cursewords).
			Show("Include curseword dictionaries?")
	}

	if isFlagSet("consolidate") {
		consolidate = defaults.consolidate
		status := pterm.FgGray.Sprint("✗ Disabled")
		if consolidate {
			status = pterm.FgGreen.Sprint("✓ Enabled")
		}
		pterm.Info.Println("Consolidated output: " + status + pterm.FgGray.Sprint(" (from --consolidate flag)"))
	} else {
		consolidate, _ = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(true).
			Show("Generate consolidated output files?")
	}

	if isFlagSet("force") {
		forceDownload = defaults.force
		status := pterm.FgGray.Sprint("✗ Use cache")
		if forceDownload {
			status = pterm.FgYellow.Sprint("✓ Enabled")
		}
		pterm.Info.Println("Force re-download: " + status + pterm.FgGray.Sprint(" (from --force flag)"))
	} else {
		forceDownload, _ = pterm.DefaultInteractiveConfirm.
			WithDefaultValue(defaults.force).
			Show("Force re-download dictionaries? (ignore cache)")
	}

	// Calculate estimates
	langList := strings.Split(langs, ",")
	numLangs := len(langList)
	numLengths := maxLen - minLen + 1

	fmt.Println()

	// Build summary using pterm panels
	pterm.DefaultSection.Println("Build Summary")

	// Configuration table
	configData := pterm.TableData{
		{"Setting", "Value"},
		{"Languages", pterm.FgCyan.Sprint(langs)},
		{"Word lengths", pterm.FgCyan.Sprintf("%d-%d characters", minLen, maxLen)},
		{"Output", pterm.FgCyan.Sprint(outDir)},
	}
	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(configData).Render()
	fmt.Println()

	// Features
	featureData := pterm.TableData{{"Feature", "Status"}}
	if ipaEnabled {
		featureData = append(featureData, []string{"IPA transcriptions", pterm.FgGreen.Sprint("✓ Enabled")})
	} else {
		featureData = append(featureData, []string{"IPA transcriptions", pterm.FgGray.Sprint("✗ Disabled")})
	}
	if cursewords {
		featureData = append(featureData, []string{"Curseword dictionaries", pterm.FgGreen.Sprint("✓ Enabled")})
	} else {
		featureData = append(featureData, []string{"Curseword dictionaries", pterm.FgGray.Sprint("✗ Disabled")})
	}
	if consolidate {
		featureData = append(featureData, []string{"Consolidated output", pterm.FgGreen.Sprint("✓ Enabled")})
	} else {
		featureData = append(featureData, []string{"Consolidated output", pterm.FgGray.Sprint("✗ Disabled")})
	}
	if forceDownload {
		featureData = append(featureData, []string{"Force re-download", pterm.FgYellow.Sprint("✓ Enabled (ignore cache)")})
	} else {
		featureData = append(featureData, []string{"Force re-download", pterm.FgGray.Sprint("✗ Use cache")})
	}
	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(featureData).Render()
	fmt.Println()

	// What will be created
	pterm.FgLightBlue.Println("Will create:")
	bulletList := pterm.BulletListItem{Level: 0, Text: fmt.Sprintf("%d language dictionaries (%d files each)", numLangs, numLengths)}
	bulletList2 := pterm.BulletListItem{Level: 0, Text: "1 synthesis dictionary (cross-language union)"}
	items := []pterm.BulletListItem{bulletList, bulletList2}
	if consolidate {
		items = append(items, pterm.BulletListItem{Level: 0, Text: "Consolidated CSV/JSON files per word length"})
		items = append(items, pterm.BulletListItem{Level: 0, Text: "all_words.json + all_words.csv (master list)"})
	}
	pterm.DefaultBulletList.WithItems(items).Render()
	fmt.Println()

	// Output structure tree
	pterm.FgLightBlue.Println("Output structure:")
	tree := pterm.TreeNode{
		Text: pterm.FgCyan.Sprint(filepath.Base(outDir) + "/"),
		Children: []pterm.TreeNode{
			{Text: "en/, tr/, ... " + pterm.FgGray.Sprint("(per-language)")},
			{Text: "synthesis/ " + pterm.FgGray.Sprint("(cross-language)")},
		},
	}
	if consolidate {
		tree.Children = append(tree.Children, pterm.TreeNode{
			Text: "../consolidated/",
			Children: []pterm.TreeNode{
				{Text: "3-c.json, 4-c.json, ..."},
				{Text: "all_words.json, all_words.csv"},
			},
		})
	}
	pterm.DefaultTree.WithRoot(tree).Render()
	fmt.Println()

	// Confirm
	proceed, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show("Proceed with build?")

	if !proceed {
		pterm.Warning.Println("Aborted.")
		os.Exit(0)
	}
	fmt.Println()

	return
}

// isFlagSet checks if a specific flag was explicitly set on command line
func isFlagSet(name string) bool {
	found := false
	pflag.Visit(func(f *pflag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// shouldSkipInteractive returns true if we should skip interactive mode entirely
func shouldSkipInteractive() bool {
	// Skip interactive if quiet or benchmark mode
	return isFlagSet("quiet") || isFlagSet("benchmark")
}

func main() {
	// Load config from config.json (falls back to hardcoded defaults)
	cfg := config.Load()

	// Flags - defaults from config.json
	languages := pflag.StringP("languages", "l", cfg.Defaults.Languages, "Comma-separated language codes")
	minLength := pflag.Int("min-length", cfg.Defaults.MinLength, "Minimum word length")
	maxLength := pflag.Int("max-length", cfg.Defaults.MaxLength, "Maximum word length")
	outputDir := pflag.StringP("output-dir", "o", "", "Output directory for dictionaries")
	cacheDir := pflag.StringP("cache-dir", "c", "", "Cache directory for downloaded sources")
	synthesis := pflag.StringP("synthesis", "s", "", "Name for synthesis dictionary")
	force := pflag.BoolP("force", "f", cfg.Defaults.Force, "Force re-download of dictionaries")
	noSplit := pflag.Bool("no-split", false, "Don't split synthesis by first letter")
	quiet := pflag.BoolP("quiet", "q", cfg.Defaults.Quiet, "Suppress progress output")
	verbose := pflag.BoolP("verbose", "v", cfg.Defaults.Verbose, "Verbose logging")
	writeMetrics := pflag.Bool("metrics", cfg.Defaults.Metrics, "Write metrics to output directory")
	benchmark := pflag.Bool("benchmark", false, "Run in benchmark mode (JSON output only)")
	consolidateOutput := pflag.Bool("consolidate", cfg.Defaults.Consolidate, "Generate consolidated output files after build")

	// Parallel processing flags
	parallel := pflag.BoolP("parallel", "p", cfg.Defaults.Parallel, "Enable parallel processing")
	workers := pflag.IntP("workers", "w", cfg.Defaults.Workers, "Number of parallel workers (0 = auto)")
	parallelIngest := pflag.Bool("parallel-ingest", cfg.Defaults.Parallel, "Enable parallel line processing within files")
	parallelBuild := pflag.Bool("parallel-build", cfg.Defaults.Parallel, "Enable parallel file writing during build")

	// Content flags
	includeCursewords := pflag.Bool("cursewords", cfg.Defaults.Cursewords, "Include curseword dictionaries")

	// Feature flags
	includeIPA := pflag.Bool("ipa", cfg.Defaults.IPA, "Generate IPA transcriptions for words")

	pflag.Parse()

	// Run interactive mode unless quiet/benchmark mode
	if !shouldSkipInteractive() {
		defaults := interactiveDefaults{
			languages:   *languages,
			minLength:   *minLength,
			maxLength:   *maxLength,
			outputDir:   *outputDir,
			ipa:         *includeIPA,
			cursewords:  *includeCursewords,
			consolidate: *consolidateOutput,
			force:       *force,
		}
		langs, minLen, maxLen, outDir, ipaOn, curse, consol, forceFlag := runInteractiveMode(defaults)
		*languages = langs
		*minLength = minLen
		*maxLength = maxLen
		*outputDir = outDir
		*includeIPA = ipaOn
		*includeCursewords = curse
		*consolidateOutput = consol
		*force = forceFlag
	}

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
		"languages":       langs,
		"min_length":      *minLength,
		"max_length":      *maxLength,
		"parallel":        *parallel,
		"workers":         *workers,
		"parallel_ingest": *parallelIngest,
		"parallel_build":  *parallelBuild,
		"cursewords":      *includeCursewords,
		"ipa":             *includeIPA,
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

			// Apply IPA transcriptions if enabled
			if *includeIPA {
				transcriber := ipa.NewTranscriber(r.Language)
				for _, word := range r.Result.Words {
					word.IPA = transcriber.Transcribe(word.Normalized)
				}
			}

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

			// Download first
			dictPath, err := ingest.Download(lang, langCacheDir, *force)
			if err != nil {
				if spinner != nil {
					spinner.Stop()
				}
				if !*benchmark {
					term.LanguageStatus(lang, "error", err.Error())
				}
				continue
			}

			// Ingest with parallel line processing if enabled
			var result *ingest.IngestResult
			if *parallelIngest {
				parseConfig := ingest.ParseConfig{
					Workers:   *workers,
					ChunkSize: 1000,
				}
				result, err = ingest.ParallelIngestHunspell(dictPath, config, parseConfig)
			} else {
				result, err = ingest.IngestHunspell(dictPath, config)
			}

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

			// Apply IPA transcriptions if enabled
			if *includeIPA {
				transcriber := ipa.NewTranscriber(lang)
				for _, word := range result.Words {
					word.IPA = transcriber.Transcribe(word.Normalized)
				}
			}

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

	// Optional: Ingest cursewords
	var cursewordCount int64
	if *includeCursewords {
		collector.StartStage("cursewords")
		if !*benchmark {
			term.Info("Including curseword dictionaries...")
		}

		for _, lang := range langs {
			if !ingest.HasCursewordSupport(lang) {
				continue
			}

			config := ingest.CursewordConfig(lang)
			config.MinLength = *minLength
			config.MaxLength = *maxLength

			langCacheDir := filepath.Join(*cacheDir, lang)
			result, err := ingest.DownloadAndIngestCursewords(lang, langCacheDir, config, *force)

			if err != nil {
				if !*benchmark {
					term.Warning(fmt.Sprintf("Cursewords [%s]: %v", lang, err))
				}
				continue
			}

			cursewordCount += int64(result.TotalValid)
			if !*benchmark {
				term.LanguageStatus(lang, "cursewords", fmt.Sprintf("%d words", result.TotalValid))
			}
			dictBuilder.AddWords(result.Words, lang)
			synthBuilder.AddWords(result.Words)
		}

		collector.EndStage("cursewords")
		collector.SetStageCounter("cursewords", "words", cursewordCount)
	}

	// Phase 2: Build per-language dictionaries
	collector.StartStage("build")
	if !*benchmark {
		term.Phase(2, 3, "Building per-language dictionaries")
	}

	var buildSpinner *ui.SpinnerWrapper
	if !*benchmark {
		buildSpinner = term.Spinner("Writing dictionary files...")
	}

	var stats *builder.BuildStats
	if *parallelBuild && *workers > 1 {
		buildConfig := builder.ParallelBuildConfig{Workers: *workers}
		stats = dictBuilder.ParallelBuild(context.Background(), buildConfig)
	} else {
		stats = dictBuilder.Build()
	}

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

	var synthStats *builder.SynthesisStats
	if *parallelBuild && *workers > 1 {
		buildConfig := builder.ParallelBuildConfig{Workers: *workers}
		synthStats = synthBuilder.ParallelBuild(context.Background(), synthConfig, buildConfig)
	} else {
		synthStats = synthBuilder.Build(synthConfig)
	}

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

	// Run consolidation if requested
	if *consolidateOutput {
		consolidatedDir := filepath.Join(filepath.Dir(*outputDir), "consolidated")
		if !*benchmark && !*quiet {
			fmt.Println()
			fmt.Println("Generating consolidated output files...")
		}
		runConsolidation(*outputDir, consolidatedDir, !*quiet && !*benchmark)
	}
}

// runConsolidation consolidates dictionary files into single output files
func runConsolidation(inputDir, outputDir string, verbose bool) {
	// Convert to absolute paths for display
	absInputDir, _ := filepath.Abs(inputDir)
	absOutputDir, _ := filepath.Abs(outputDir)

	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		return
	}

	type wordEntry struct {
		Normalized string   `json:"normalized"`
		IPA        string   `json:"ipa,omitempty"`
		Languages  []string `json:"languages"`
	}

	type dictFile struct {
		WordType string               `json:"word_type"`
		Words    map[string]wordEntry `json:"words"`
	}

	// Collect words by type
	wordsByType := make(map[string]map[string]wordEntry)

	filepath.Walk(absInputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Skip config and metrics files
		name := info.Name()
		if strings.HasPrefix(name, "_") ||
			strings.HasPrefix(name, "run_") ||
			name == "latest.json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var dict dictFile
		if err := json.Unmarshal(data, &dict); err != nil {
			return nil
		}

		// Skip files without words (likely metrics or config files)
		if dict.Words == nil || len(dict.Words) == 0 {
			return nil
		}

		wordType := dict.WordType
		if wordType == "" {
			wordType = strings.TrimSuffix(name, ".json")
		}

		if wordsByType[wordType] == nil {
			wordsByType[wordType] = make(map[string]wordEntry)
		}

		for key, word := range dict.Words {
			if _, exists := wordsByType[wordType][key]; !exists {
				wordsByType[wordType][key] = word
			}
		}
		return nil
	})

	// Sort types
	types := make([]string, 0, len(wordsByType))
	for t := range wordsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	allWords := make([]string, 0)

	for _, wordType := range types {
		words := wordsByType[wordType]

		sortedWords := make([]string, 0, len(words))
		for w := range words {
			sortedWords = append(sortedWords, w)
		}
		sort.Strings(sortedWords)

		// JSON output
		output := struct {
			Type  string   `json:"type"`
			Count int      `json:"count"`
			Words []string `json:"words"`
		}{
			Type:  wordType,
			Count: len(sortedWords),
			Words: sortedWords,
		}

		jsonPath := filepath.Join(absOutputDir, wordType+".json")
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		os.WriteFile(jsonPath, jsonData, 0644)

		// CSV output
		csvPath := filepath.Join(absOutputDir, wordType+".csv")
		csvFile, _ := os.Create(csvPath)
		csvFile.WriteString("word,ipa,languages\n")
		for _, w := range sortedWords {
			entry := words[w]
			langs := strings.Join(entry.Languages, ";")
			csvFile.WriteString(fmt.Sprintf("%s,%s,%s\n", w, entry.IPA, langs))
		}
		csvFile.Close()

		allWords = append(allWords, sortedWords...)

		if verbose {
			fmt.Printf("  %s: %d words -> %s\n", wordType, len(sortedWords), jsonPath)
		}
	}

	// Deduplicate all words
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, w := range allWords {
		if !seen[w] {
			seen[w] = true
			unique = append(unique, w)
		}
	}
	sort.Strings(unique)

	// Write all_words files
	allJSON := struct {
		Type  string   `json:"type"`
		Count int      `json:"count"`
		Words []string `json:"words"`
	}{
		Type:  "all",
		Count: len(unique),
		Words: unique,
	}
	jsonData, _ := json.MarshalIndent(allJSON, "", "  ")
	allWordsJSON := filepath.Join(absOutputDir, "all_words.json")
	os.WriteFile(allWordsJSON, jsonData, 0644)

	allWordsCSV := filepath.Join(absOutputDir, "all_words.csv")
	csvFile, _ := os.Create(allWordsCSV)
	for _, w := range unique {
		csvFile.WriteString(w + "\n")
	}
	csvFile.Close()

	if verbose {
		fmt.Println()
		pterm.Info.Println("Consolidated output:")
		fmt.Printf("  Total unique: %d words\n", len(unique))
		fmt.Printf("  JSON: %s\n", allWordsJSON)
		fmt.Printf("  CSV:  %s\n", allWordsCSV)
		fmt.Printf("  Dir:  %s\n", absOutputDir)
	}
}
