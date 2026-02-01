package ingest

import (
	"fmt"
	"os"
	"sync"
)

// ParallelConfig configures parallel processing behavior.
type ParallelConfig struct {
	Workers   int  // Number of parallel workers (0 = sequential)
	Force     bool // Force re-download
	MinLength int
	MaxLength int
}

// LanguageResult holds the result for a single language.
type LanguageResult struct {
	Language string
	Result   *IngestResult
	Error    error
	Cached   bool
}

// ProgressCallback is called when a language completes processing.
type ProgressCallback func(lang string, result *LanguageResult)

// ParallelDownloadAndIngest downloads and ingests multiple languages in parallel.
func ParallelDownloadAndIngest(
	languages []string,
	cacheDir string,
	config ParallelConfig,
	callback ProgressCallback,
) []*LanguageResult {
	results := make([]*LanguageResult, len(languages))

	if config.Workers <= 1 {
		// Sequential processing
		for i, lang := range languages {
			result := downloadAndIngestLanguage(lang, cacheDir, config)
			results[i] = result
			if callback != nil {
				callback(lang, result)
			}
		}
		return results
	}

	// Parallel processing with worker pool
	type job struct {
		index    int
		language string
	}

	jobs := make(chan job, len(languages))
	resultsChan := make(chan struct {
		index  int
		result *LanguageResult
	}, len(languages))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < config.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				result := downloadAndIngestLanguage(j.language, cacheDir, config)
				resultsChan <- struct {
					index  int
					result *LanguageResult
				}{j.index, result}
			}
		}()
	}

	// Send jobs
	go func() {
		for i, lang := range languages {
			jobs <- job{i, lang}
		}
		close(jobs)
	}()

	// Collect results in a separate goroutine
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Process results as they come in
	for r := range resultsChan {
		results[r.index] = r.result
		if callback != nil {
			callback(r.result.Language, r.result)
		}
	}

	return results
}

// downloadAndIngestLanguage handles a single language.
func downloadAndIngestLanguage(language, cacheDir string, config ParallelConfig) *LanguageResult {
	langCacheDir := fmt.Sprintf("%s/%s", cacheDir, language)

	ingestConfig := IngestConfig{
		Language:  language,
		Category:  "standard",
		MinLength: config.MinLength,
		MaxLength: config.MaxLength,
	}

	// Check if cached first (to set Cached flag)
	cached := false
	if !config.Force {
		cachedPath := fmt.Sprintf("%s/%s.dic", langCacheDir, language)
		if fileExists(cachedPath) {
			cached = true
		}
	}

	result, err := DownloadAndIngest(language, langCacheDir, ingestConfig, config.Force)

	return &LanguageResult{
		Language: language,
		Result:   result,
		Error:    err,
		Cached:   cached,
	}
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ParallelStats holds aggregate statistics from parallel processing.
type ParallelStats struct {
	TotalLanguages int
	Successful     int
	Failed         int
	Cached         int
	TotalRaw       int
	TotalValid     int
	TotalDuplicates int
}

// AggregateResults computes statistics from parallel results.
func AggregateResults(results []*LanguageResult) *ParallelStats {
	stats := &ParallelStats{
		TotalLanguages: len(results),
	}

	for _, r := range results {
		if r.Error != nil {
			stats.Failed++
			continue
		}
		stats.Successful++
		if r.Cached {
			stats.Cached++
		}
		if r.Result != nil {
			stats.TotalRaw += r.Result.TotalRaw
			stats.TotalValid += r.Result.TotalValid
			stats.TotalDuplicates += r.Result.TotalDuplicates
		}
	}

	return stats
}
