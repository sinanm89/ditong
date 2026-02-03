package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"ditong/internal/schema"
)

// ParallelBuildConfig configures parallel build operations.
type ParallelBuildConfig struct {
	Workers int // Number of parallel workers for file writing
}

// DefaultParallelBuildConfig returns sensible defaults.
func DefaultParallelBuildConfig() ParallelBuildConfig {
	return ParallelBuildConfig{
		Workers: 4,
	}
}

// writeJob represents a file write job.
type writeJob struct {
	filePath   string
	dictionary *schema.Dictionary
}

// ParallelBuild builds all dictionary files concurrently.
// Accepts a context for cancellation support.
func (b *DictionaryBuilder) ParallelBuild(ctx context.Context, config ParallelBuildConfig) *BuildStats {
	stats := NewBuildStats()

	if config.Workers <= 1 {
		return b.Build() // Fall back to sequential
	}

	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return stats
	default:
	}

	// Collect all write jobs
	var jobs []writeJob
	var mu sync.Mutex

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
			jobs = append(jobs, writeJob{filePath: filePath, dictionary: dictionary})
		}
	}

	// Process jobs in parallel
	jobsChan := make(chan writeJob, len(jobs))
	var wg sync.WaitGroup

	for w := 0; w < config.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				select {
				case <-ctx.Done():
					return
				default:
					if err := job.dictionary.Save(job.filePath); err == nil {
						mu.Lock()
						stats.FilesWritten = append(stats.FilesWritten, job.filePath)
						mu.Unlock()
					}
				}
			}
		}()
	}

	// Send jobs (check context between sends)
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			close(jobsChan)
			wg.Wait()
			return stats
		case jobsChan <- job:
		}
	}
	close(jobsChan)
	wg.Wait()

	return stats
}

// synthWriteJob represents a synthesis file write job.
type synthWriteJob struct {
	filePath   string
	dictionary *schema.Dictionary
}

// ParallelBuild builds a synthesis dictionary with concurrent file writing.
// Accepts a context for cancellation support.
func (b *SynthesisBuilder) ParallelBuild(ctx context.Context, config *SynthesisConfig, parallelConfig ParallelBuildConfig) *SynthesisStats {
	if parallelConfig.Workers <= 1 {
		return b.Build(config) // Fall back to sequential
	}

	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return &SynthesisStats{ConfigName: config.Name}
	default:
	}

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

	// Prepare output directory
	synthDir := filepath.Join(b.OutputDir, config.Name)
	os.MkdirAll(synthDir, 0755)

	// Write config metadata (sequential - single file)
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

	// Collect write jobs
	var jobs []synthWriteJob
	var mu sync.Mutex

	for length, letterDict := range filtered {
		if len(letterDict) == 0 {
			continue
		}

		wordType := fmt.Sprintf("%d-c", length)

		if config.SplitByLetter {
			lengthDir := filepath.Join(synthDir, wordType)
			os.MkdirAll(lengthDir, 0755)

			for letter, words := range letterDict {
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
				jobs = append(jobs, synthWriteJob{filePath: filePath, dictionary: dictionary})
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
			jobs = append(jobs, synthWriteJob{filePath: filePath, dictionary: dictionary})
		}
	}

	// Process jobs in parallel
	jobsChan := make(chan synthWriteJob, len(jobs))
	var wg sync.WaitGroup

	for w := 0; w < parallelConfig.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				select {
				case <-ctx.Done():
					return
				default:
					if err := job.dictionary.Save(job.filePath); err == nil {
						mu.Lock()
						stats.FilesWritten = append(stats.FilesWritten, job.filePath)
						mu.Unlock()
					}
				}
			}
		}()
	}

	// Send jobs (check context between sends)
	for _, job := range jobs {
		select {
		case <-ctx.Done():
			close(jobsChan)
			wg.Wait()
			return stats
		case jobsChan <- job:
		}
	}
	close(jobsChan)
	wg.Wait()

	return stats
}
