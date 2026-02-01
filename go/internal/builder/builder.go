// Package builder provides dictionary building and synthesis functionality.
package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"ditong/internal/schema"
)

// BuildStats holds statistics from a build operation.
type BuildStats struct {
	TotalWords   int
	ByLength     map[int]int
	ByLanguage   map[string]int
	ByCategory   map[string]int
	FilesWritten []string
}

// NewBuildStats creates a new BuildStats.
func NewBuildStats() *BuildStats {
	return &BuildStats{
		ByLength:   make(map[int]int),
		ByLanguage: make(map[string]int),
		ByCategory: make(map[string]int),
	}
}

// DictionaryBuilder builds organized dictionary files.
type DictionaryBuilder struct {
	OutputDir string
	MinLength int
	MaxLength int
	words     map[string]map[int]map[string]*schema.Word // lang -> length -> normalized -> Word
}

// NewDictionaryBuilder creates a new DictionaryBuilder.
func NewDictionaryBuilder(outputDir string, minLength, maxLength int) *DictionaryBuilder {
	return &DictionaryBuilder{
		OutputDir: outputDir,
		MinLength: minLength,
		MaxLength: maxLength,
		words:     make(map[string]map[int]map[string]*schema.Word),
	}
}

// AddWords adds words from an ingest result.
func (b *DictionaryBuilder) AddWords(words []*schema.Word, language string) {
	if _, ok := b.words[language]; !ok {
		b.words[language] = make(map[int]map[string]*schema.Word)
		for i := b.MinLength; i <= b.MaxLength; i++ {
			b.words[language][i] = make(map[string]*schema.Word)
		}
	}

	for _, word := range words {
		if word.Length < b.MinLength || word.Length > b.MaxLength {
			continue
		}

		lengthDict := b.words[language][word.Length]
		if existing, ok := lengthDict[word.Normalized]; ok {
			for _, source := range word.Sources {
				existing.AddSource(source)
			}
			for tag := range word.Tags {
				existing.Tags[tag] = true
			}
		} else {
			lengthDict[word.Normalized] = word
		}
	}
}

// Build builds all dictionary files.
func (b *DictionaryBuilder) Build() *BuildStats {
	stats := NewBuildStats()

	for language, lengthDicts := range b.words {
		langDir := filepath.Join(b.OutputDir, language)
		if err := os.MkdirAll(langDir, 0755); err != nil {
			continue
		}

		for length, wordsDict := range lengthDicts {
			if len(wordsDict) == 0 {
				continue
			}

			wordType := fmt.Sprintf("%d-c", length)
			dictionary := schema.NewDictionary(fmt.Sprintf("%s_%s", language, wordType))
			dictionary.Language = language
			dictionary.WordType = wordType

			for _, word := range wordsDict {
				dictionary.AddWord(word)
				stats.TotalWords++
				stats.ByLength[length]++
				stats.ByLanguage[language]++
				for cat := range word.Categories {
					stats.ByCategory[cat]++
				}
			}

			filePath := filepath.Join(langDir, fmt.Sprintf("%s.json", wordType))
			if err := dictionary.Save(filePath); err == nil {
				stats.FilesWritten = append(stats.FilesWritten, filePath)
			}
		}
	}

	return stats
}

// SynthesisConfig configures a synthesis build.
type SynthesisConfig struct {
	Name              string
	IncludeLanguages  map[string]bool
	ExcludeLanguages  map[string]bool
	IncludeCategories map[string]bool
	ExcludeCategories map[string]bool
	MinLength         int
	MaxLength         int
	SplitByLetter     bool
}

// NewSynthesisConfig creates a new SynthesisConfig.
func NewSynthesisConfig(name string) *SynthesisConfig {
	return &SynthesisConfig{
		Name:              name,
		IncludeLanguages:  make(map[string]bool),
		IncludeCategories: make(map[string]bool),
		ExcludeCategories: make(map[string]bool),
		MinLength:         3,
		MaxLength:         10,
		SplitByLetter:     true,
	}
}

// SynthesisStats holds statistics from a synthesis build.
type SynthesisStats struct {
	ConfigName         string
	TotalWords         int
	ByLength           map[int]int
	ByLetter           map[string]int
	LanguagesIncluded  map[string]bool
	CategoriesIncluded map[string]bool
	FilesWritten       []string
}

// SynthesisBuilder builds synthesis dictionaries.
type SynthesisBuilder struct {
	OutputDir string
	wordPool  map[string]*schema.Word // normalized -> Word
}

// NewSynthesisBuilder creates a new SynthesisBuilder.
func NewSynthesisBuilder(outputDir string) *SynthesisBuilder {
	return &SynthesisBuilder{
		OutputDir: filepath.Join(outputDir, "synthesis"),
		wordPool:  make(map[string]*schema.Word),
	}
}

// AddWords adds words to the pool.
func (b *SynthesisBuilder) AddWords(words []*schema.Word) {
	for _, word := range words {
		if existing, ok := b.wordPool[word.Normalized]; ok {
			for _, source := range word.Sources {
				existing.AddSource(source)
			}
			for tag := range word.Tags {
				existing.Tags[tag] = true
			}
		} else {
			b.wordPool[word.Normalized] = word
		}
	}
}

// Build builds a synthesis dictionary.
func (b *SynthesisBuilder) Build(config *SynthesisConfig) *SynthesisStats {
	stats := &SynthesisStats{
		ConfigName:         config.Name,
		ByLength:           make(map[int]int),
		ByLetter:           make(map[string]int),
		LanguagesIncluded:  make(map[string]bool),
		CategoriesIncluded: make(map[string]bool),
	}

	// Filter words
	filtered := make(map[int]map[string][]*schema.Word)
	for length := config.MinLength; length <= config.MaxLength; length++ {
		filtered[length] = make(map[string][]*schema.Word)
	}

	for _, word := range b.wordPool {
		if !word.MatchesFilter(
			config.IncludeCategories,
			config.ExcludeCategories,
			config.IncludeLanguages,
			config.MinLength,
			config.MaxLength,
		) {
			continue
		}

		letter := string(word.Normalized[0])
		filtered[word.Length][letter] = append(filtered[word.Length][letter], word)
		stats.TotalWords++
		stats.ByLength[word.Length]++
		stats.ByLetter[letter]++
		for lang := range word.Languages {
			stats.LanguagesIncluded[lang] = true
		}
		for cat := range word.Categories {
			stats.CategoriesIncluded[cat] = true
		}
	}

	// Write output
	synthDir := filepath.Join(b.OutputDir, config.Name)
	os.MkdirAll(synthDir, 0755)

	// Write config metadata
	configFile := filepath.Join(synthDir, "_config.json")
	metadata := map[string]interface{}{
		"config": map[string]interface{}{
			"name":       config.Name,
			"min_length": config.MinLength,
			"max_length": config.MaxLength,
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"stats": map[string]interface{}{
			"total_words": stats.TotalWords,
			"by_length":   stats.ByLength,
		},
	}

	if f, err := os.Create(configFile); err == nil {
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		enc.Encode(metadata)
		f.Close()
		stats.FilesWritten = append(stats.FilesWritten, configFile)
	}

	// Write word files
	for length, letterDict := range filtered {
		if len(letterDict) == 0 {
			continue
		}

		wordType := fmt.Sprintf("%d-c", length)

		if config.SplitByLetter {
			lengthDir := filepath.Join(synthDir, wordType)
			os.MkdirAll(lengthDir, 0755)

			letters := make([]string, 0, len(letterDict))
			for letter := range letterDict {
				letters = append(letters, letter)
			}
			sort.Strings(letters)

			for _, letter := range letters {
				words := letterDict[letter]
				dictionary := schema.NewDictionary(
					fmt.Sprintf("%s_%s_%s", config.Name, wordType, letter),
				)
				dictionary.WordType = wordType

				sort.Slice(words, func(i, j int) bool {
					return words[i].Normalized < words[j].Normalized
				})

				for _, word := range words {
					dictionary.AddWord(word)
				}

				filePath := filepath.Join(lengthDir, fmt.Sprintf("%s.json", letter))
				if err := dictionary.Save(filePath); err == nil {
					stats.FilesWritten = append(stats.FilesWritten, filePath)
				}
			}
		} else {
			dictionary := schema.NewDictionary(
				fmt.Sprintf("%s_%s", config.Name, wordType),
			)
			dictionary.WordType = wordType

			for _, words := range letterDict {
				for _, word := range words {
					dictionary.AddWord(word)
				}
			}

			filePath := filepath.Join(synthDir, fmt.Sprintf("%s.json", wordType))
			if err := dictionary.Save(filePath); err == nil {
				stats.FilesWritten = append(stats.FilesWritten, filePath)
			}
		}
	}

	return stats
}
