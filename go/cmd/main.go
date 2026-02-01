// ditong CLI - Multi-language lexicon toolkit.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ditong/internal/builder"
	"ditong/internal/ingest"
)

func main() {
	languages := flag.String("languages", "en,tr", "Comma-separated language codes")
	minLength := flag.Int("min-length", 3, "Minimum word length")
	maxLength := flag.Int("max-length", 10, "Maximum word length")
	outputDir := flag.String("output-dir", "", "Output directory for dictionaries")
	cacheDir := flag.String("cache-dir", "", "Cache directory for downloaded sources")
	synthesis := flag.String("synthesis", "", "Name for synthesis dictionary")
	force := flag.Bool("force", false, "Force re-download of dictionaries")
	noSplit := flag.Bool("no-split", false, "Don't split synthesis by first letter")

	flag.Parse()

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

	langs := strings.Split(*languages, ",")
	for i := range langs {
		langs[i] = strings.TrimSpace(langs[i])
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("ditong - Multi-language Lexicon Toolkit")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Languages: %s\n", strings.Join(langs, ", "))
	fmt.Printf("Length range: %d-%d\n", *minLength, *maxLength)
	fmt.Printf("Output: %s\n", *outputDir)
	fmt.Println()

	// Initialize builders
	dictBuilder := builder.NewDictionaryBuilder(*outputDir, *minLength, *maxLength)
	synthBuilder := builder.NewSynthesisBuilder(*outputDir)

	// Ingest each language
	fmt.Println("[1/3] Downloading and ingesting dictionaries...")
	for _, lang := range langs {
		fmt.Printf("\n  [%s] ", lang)

		config := ingest.DefaultConfig(lang)
		config.MinLength = *minLength
		config.MaxLength = *maxLength

		langCacheDir := filepath.Join(*cacheDir, lang)
		result, err := ingest.DownloadAndIngest(lang, langCacheDir, config, *force)
		if err != nil {
			fmt.Printf("ERROR - %v\n", err)
			continue
		}

		fmt.Printf("OK - %d words\n", result.TotalValid)
		dictBuilder.AddWords(result.Words, lang)
		synthBuilder.AddWords(result.Words)
	}

	// Build per-language dictionaries
	fmt.Println("\n[2/3] Building per-language dictionaries...")
	stats := dictBuilder.Build()
	fmt.Printf("  Total words: %d\n", stats.TotalWords)
	fmt.Printf("  Files written: %d\n", len(stats.FilesWritten))

	// Sort languages for display
	langKeys := make([]string, 0, len(stats.ByLanguage))
	for lang := range stats.ByLanguage {
		langKeys = append(langKeys, lang)
	}
	sort.Strings(langKeys)

	for _, lang := range langKeys {
		fmt.Printf("    %s: %d\n", lang, stats.ByLanguage[lang])
	}

	// Build synthesis dictionary
	fmt.Println("\n[3/3] Building synthesis dictionary...")
	synthName := *synthesis
	if synthName == "" {
		sort.Strings(langs)
		synthName = strings.Join(langs, "_") + "_standard"
	}

	synthConfig := builder.NewSynthesisConfig(synthName)
	for _, lang := range langs {
		synthConfig.IncludeLanguages[lang] = true
	}
	synthConfig.IncludeCategories["standard"] = true
	synthConfig.MinLength = *minLength
	synthConfig.MaxLength = *maxLength
	synthConfig.SplitByLetter = !*noSplit

	synthStats := synthBuilder.Build(synthConfig)
	fmt.Printf("  Synthesis name: %s\n", synthName)
	fmt.Printf("  Unique words: %d\n", synthStats.TotalWords)
	fmt.Printf("  Files written: %d\n", len(synthStats.FilesWritten))

	fmt.Println("\n  By length:")
	lengthKeys := make([]int, 0, len(synthStats.ByLength))
	for length := range synthStats.ByLength {
		lengthKeys = append(lengthKeys, length)
	}
	sort.Ints(lengthKeys)

	for _, length := range lengthKeys {
		fmt.Printf("    %d-c: %d\n", length, synthStats.ByLength[length])
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Done!")
	fmt.Println(strings.Repeat("=", 60))
}
