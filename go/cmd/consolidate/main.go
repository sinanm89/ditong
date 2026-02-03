// ditong-consolidate - Consolidate dictionary output to single files
// Usage: ditong-consolidate -i test_output -o output/consolidated
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/pflag"
)

type Word struct {
	Normalized string   `json:"normalized"`
	Length     int      `json:"length"`
	Type       string   `json:"type"`
	IPA        string   `json:"ipa,omitempty"`
	Languages  []string `json:"languages"`
	Categories []string `json:"categories"`
}

type Dictionary struct {
	Name      string          `json:"name"`
	WordType  string          `json:"word_type"`
	WordCount int             `json:"word_count"`
	Words     map[string]Word `json:"words"`
}

type ConsolidatedOutput struct {
	Type      string   `json:"type"`
	Count     int      `json:"count"`
	Words     []string `json:"words"`
	WithIPA   []WordSimple `json:"words_with_metadata,omitempty"`
}

type WordSimple struct {
	Word      string   `json:"word"`
	IPA       string   `json:"ipa,omitempty"`
	Languages []string `json:"languages"`
}

func main() {
	inputDir := pflag.StringP("input", "i", "", "Input directory with dictionary files")
	outputDir := pflag.StringP("output", "o", "output/consolidated", "Output directory")
	withMeta := pflag.BoolP("metadata", "m", false, "Include IPA and language metadata")
	pflag.Parse()

	if *inputDir == "" {
		fmt.Fprintln(os.Stderr, "Usage: ditong-consolidate -i <input-dir> [-o <output-dir>]")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		os.Exit(1)
	}

	// Collect words by type
	wordsByType := make(map[string]map[string]Word)

	err := filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		if strings.HasPrefix(info.Name(), "_") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var dict Dictionary
		if err := json.Unmarshal(data, &dict); err != nil {
			return nil
		}

		if dict.Words == nil {
			return nil
		}

		wordType := dict.WordType
		if wordType == "" {
			// Extract from filename (e.g., "3-c.json" -> "3-c")
			wordType = strings.TrimSuffix(info.Name(), ".json")
		}

		if wordsByType[wordType] == nil {
			wordsByType[wordType] = make(map[string]Word)
		}

		for key, word := range dict.Words {
			if _, exists := wordsByType[wordType][key]; !exists {
				wordsByType[wordType][key] = word
			}
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Get sorted types
	types := make([]string, 0, len(wordsByType))
	for t := range wordsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	allWords := make([]string, 0)

	fmt.Printf("Consolidating dictionaries from %s\n\n", *inputDir)

	for _, wordType := range types {
		words := wordsByType[wordType]

		// Sort words
		sortedWords := make([]string, 0, len(words))
		for w := range words {
			sortedWords = append(sortedWords, w)
		}
		sort.Strings(sortedWords)

		// JSON output
		output := ConsolidatedOutput{
			Type:  wordType,
			Count: len(sortedWords),
			Words: sortedWords,
		}

		if *withMeta {
			output.WithIPA = make([]WordSimple, 0, len(sortedWords))
			for _, w := range sortedWords {
				output.WithIPA = append(output.WithIPA, WordSimple{
					Word:      w,
					IPA:       words[w].IPA,
					Languages: words[w].Languages,
				})
			}
		}

		jsonPath := filepath.Join(*outputDir, wordType+".json")
		jsonData, _ := json.MarshalIndent(output, "", "  ")
		os.WriteFile(jsonPath, jsonData, 0644)

		// CSV output
		csvPath := filepath.Join(*outputDir, wordType+".csv")
		csvFile, _ := os.Create(csvPath)
		writer := csv.NewWriter(csvFile)

		if *withMeta {
			writer.Write([]string{"word", "ipa", "languages"})
			for _, w := range sortedWords {
				writer.Write([]string{
					w,
					words[w].IPA,
					strings.Join(words[w].Languages, ";"),
				})
			}
		} else {
			for _, w := range sortedWords {
				writer.Write([]string{w})
			}
		}
		writer.Flush()
		csvFile.Close()

		allWords = append(allWords, sortedWords...)
		fmt.Printf("  %s: %d words\n", wordType, len(sortedWords))
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
	allJSON := ConsolidatedOutput{
		Type:  "all",
		Count: len(unique),
		Words: unique,
	}
	jsonData, _ := json.MarshalIndent(allJSON, "", "  ")
	os.WriteFile(filepath.Join(*outputDir, "all_words.json"), jsonData, 0644)

	csvFile, _ := os.Create(filepath.Join(*outputDir, "all_words.csv"))
	writer := csv.NewWriter(csvFile)
	for _, w := range unique {
		writer.Write([]string{w})
	}
	writer.Flush()
	csvFile.Close()

	fmt.Printf("\n  Total unique: %d words\n", len(unique))
	fmt.Printf("\nOutput: %s/\n", *outputDir)
}
