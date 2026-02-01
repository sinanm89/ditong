// ditong CLI - Multi-language lexicon toolkit.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ditong/internal/builder"
	"ditong/internal/ingest"
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

	pflag.Parse()

	// Initialize UI
	term := ui.New(*quiet, *verbose)

	// Print banner
	term.Banner()

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

	// Show configuration
	term.Config(langs, *minLength, *maxLength, *outputDir)

	startTime := time.Now()

	// Initialize builders
	dictBuilder := builder.NewDictionaryBuilder(*outputDir, *minLength, *maxLength)
	synthBuilder := builder.NewSynthesisBuilder(*outputDir)

	// Phase 1: Download and ingest
	term.Phase(1, 3, "Downloading and ingesting dictionaries")

	for _, lang := range langs {
		spinner := term.Spinner(fmt.Sprintf("Processing %s...", strings.ToUpper(lang)))

		config := ingest.DefaultConfig(lang)
		config.MinLength = *minLength
		config.MaxLength = *maxLength

		langCacheDir := filepath.Join(*cacheDir, lang)
		result, err := ingest.DownloadAndIngest(lang, langCacheDir, config, *force)

		spinner.Stop()

		if err != nil {
			term.LanguageStatus(lang, "error", err.Error())
			continue
		}

		term.LanguageStatus(lang, "ok", fmt.Sprintf("%d words ingested", result.TotalValid))
		dictBuilder.AddWords(result.Words, lang)
		synthBuilder.AddWords(result.Words)
	}

	// Phase 2: Build per-language dictionaries
	term.Phase(2, 3, "Building per-language dictionaries")

	buildSpinner := term.Spinner("Writing dictionary files...")
	stats := dictBuilder.Build()
	buildSpinner.Stop()

	term.LanguageStats(stats.ByLanguage)
	term.Info(fmt.Sprintf("Total: %d words in %d files", stats.TotalWords, len(stats.FilesWritten)))

	// Phase 3: Build synthesis dictionary
	term.Phase(3, 3, "Building synthesis dictionary")

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

	synthSpinner := term.Spinner(fmt.Sprintf("Building synthesis: %s...", synthName))
	synthStats := synthBuilder.Build(synthConfig)
	synthSpinner.Stop()

	term.Info(fmt.Sprintf("Synthesis: %s", synthName))
	term.LengthStats(synthStats.ByLength)
	term.Info(fmt.Sprintf("Unique words: %d in %d files", synthStats.TotalWords, len(synthStats.FilesWritten)))

	// Final report
	duration := time.Since(startTime)
	totalFiles := len(stats.FilesWritten) + len(synthStats.FilesWritten)
	term.FinalReport(stats.TotalWords, totalFiles, duration)
	term.Done()
}
