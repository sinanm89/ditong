package ingest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ditong/internal/normalizer"
	"ditong/internal/schema"
)

// ParseConfig configures parallel file parsing.
type ParseConfig struct {
	Workers   int // Number of parallel workers for line processing
	ChunkSize int // Lines per chunk (0 = auto)
}

// DefaultParseConfig returns sensible defaults.
func DefaultParseConfig() ParseConfig {
	return ParseConfig{
		Workers:   4,
		ChunkSize: 1000,
	}
}

// lineChunk represents a chunk of lines to process.
type lineChunk struct {
	lines      []string
	startLine  int
	dictName   string
	absPath    string
	language   string
	category   string
	minLength  int
	maxLength  int
}

// chunkResult holds processed words from a chunk.
type chunkResult struct {
	words      map[string]*schema.Word
	rawCount   int
	dupCount   int
}

// ParallelIngestHunspell ingests a Hunspell dictionary with parallel line processing.
func ParallelIngestHunspell(filePath string, config IngestConfig, parseConfig ParseConfig) (*IngestResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	absPath, _ := os.Getwd()
	if ap, err := filepath.Abs(filePath); err == nil {
		absPath = ap
	}
	dictName := "hunspell_" + config.Language

	// Read all lines into memory
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Skip first line if it's a word count
	startIdx := 0
	if len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		isDigits := true
		for _, r := range first {
			if r < '0' || r > '9' {
				isDigits = false
				break
			}
		}
		if isDigits {
			startIdx = 1
		}
	}

	// Sequential fallback for small files or single worker
	if parseConfig.Workers <= 1 || len(lines)-startIdx < parseConfig.ChunkSize*2 {
		return IngestHunspell(filePath, config)
	}

	// Split into chunks
	chunkSize := parseConfig.ChunkSize
	if chunkSize <= 0 {
		chunkSize = (len(lines) - startIdx) / parseConfig.Workers
		if chunkSize < 100 {
			chunkSize = 100
		}
	}

	var chunks []lineChunk
	for i := startIdx; i < len(lines); i += chunkSize {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		chunks = append(chunks, lineChunk{
			lines:     lines[i:end],
			startLine: i + 1, // 1-indexed
			dictName:  dictName,
			absPath:   absPath,
			language:  config.Language,
			category:  config.Category,
			minLength: config.MinLength,
			maxLength: config.MaxLength,
		})
	}

	// Process chunks in parallel
	results := make([]chunkResult, len(chunks))
	var wg sync.WaitGroup

	// Create worker pool
	jobs := make(chan int, len(chunks))
	for w := 0; w < parseConfig.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				results[idx] = processChunk(chunks[idx])
			}
		}()
	}

	// Send jobs
	for i := range chunks {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	// Merge results
	merged := make(map[string]*schema.Word)
	totalRaw := 0
	totalDup := 0

	for _, r := range results {
		totalRaw += r.rawCount
		for norm, word := range r.words {
			if existing, ok := merged[norm]; ok {
				for _, src := range word.Sources {
					existing.AddSource(src)
				}
				totalDup++
			} else {
				merged[norm] = word
			}
		}
	}

	// Build final result
	finalWords := make([]*schema.Word, 0, len(merged))
	for _, w := range merged {
		finalWords = append(finalWords, w)
	}

	return &IngestResult{
		Words:           finalWords,
		SourcePath:      absPath,
		DictName:        dictName,
		Language:        config.Language,
		Category:        config.Category,
		TotalRaw:        totalRaw,
		TotalValid:      len(merged),
		TotalDuplicates: totalDup,
	}, nil
}

// processChunk processes a single chunk of lines.
func processChunk(chunk lineChunk) chunkResult {
	result := chunkResult{
		words: make(map[string]*schema.Word),
	}

	for i, line := range chunk.lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result.rawCount++

		// Strip affix flags
		word := line
		if idx := strings.Index(line, "/"); idx != -1 {
			word = line[:idx]
		}
		if word == "" {
			continue
		}

		normalized := normalizer.NormalizeAndValidate(word)
		if normalized == "" {
			continue
		}

		length := len(normalized)
		if length < chunk.minLength || length > chunk.maxLength {
			continue
		}

		lineNum := chunk.startLine + i
		source := schema.WordSource{
			DictName:     chunk.dictName,
			DictFilepath: chunk.absPath,
			Language:     chunk.language,
			OriginalForm: word,
			LineNumber:   &lineNum,
			Category:     chunk.category,
		}

		if existing, ok := result.words[normalized]; ok {
			existing.AddSource(source)
			result.dupCount++
		} else {
			w := schema.NewWord(normalized, length, wordTypeFromLength(length))
			w.AddSource(source)
			result.words[normalized] = w
		}
	}

	return result
}

// wordTypeFromLength returns the word type string for a given length.
func wordTypeFromLength(length int) string {
	if length >= 3 && length <= 10 {
		return fmt.Sprintf("%d-c", length)
	}
	return ""
}
