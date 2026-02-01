// Package ingest provides dictionary ingestion from various sources.
package ingest

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ditong/internal/normalizer"
	"ditong/internal/schema"
)

// HunspellURLs maps language codes to Hunspell dictionary URLs.
var HunspellURLs = map[string]string{
	"en": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/en/index.dic",
	"tr": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/tr/index.dic",
	"de": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/de/index.dic",
	"fr": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/fr/index.dic",
	"es": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/es/index.dic",
	"it": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/it/index.dic",
	"pt": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/pt/index.dic",
	"nl": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/nl/index.dic",
	"pl": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/pl/index.dic",
	"ru": "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/ru/index.dic",
}

// IngestResult holds the result of ingesting a dictionary.
type IngestResult struct {
	Words           []*schema.Word
	SourcePath      string
	DictName        string
	Language        string
	Category        string
	TotalRaw        int
	TotalValid      int
	TotalDuplicates int
	Errors          []string
}

// IngestConfig configures ingestion behavior.
type IngestConfig struct {
	Language  string
	Category  string
	MinLength int
	MaxLength int
}

// DefaultConfig returns default ingestion config.
func DefaultConfig(language string) IngestConfig {
	return IngestConfig{
		Language:  language,
		Category:  "standard",
		MinLength: 3,
		MaxLength: 10,
	}
}

// Download downloads a Hunspell dictionary to the cache directory.
func Download(language, cacheDir string, force bool) (string, error) {
	url, ok := HunspellURLs[language]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", language)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	filename := fmt.Sprintf("%s.dic", language)
	cachedPath := filepath.Join(cacheDir, filename)

	if !force {
		if _, err := os.Stat(cachedPath); err == nil {
			fmt.Printf("[%s] Using cached: %s\n", language, cachedPath)
			return cachedPath, nil
		}
	}

	fmt.Printf("[%s] Downloading from: %s\n", language, url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	file, err := os.Create(cachedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("[%s] Saved to: %s\n", language, cachedPath)
	return cachedPath, nil
}

// IngestHunspell ingests a Hunspell dictionary file.
func IngestHunspell(filePath string, config IngestConfig) (*IngestResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	absPath, _ := filepath.Abs(filePath)
	dictName := fmt.Sprintf("hunspell_%s", config.Language)

	result := &IngestResult{
		SourcePath: absPath,
		DictName:   dictName,
		Language:   config.Language,
		Category:   config.Category,
	}

	words := make(map[string]*schema.Word)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Skip first line if it's a word count
		if lineNum == 1 {
			isDigits := true
			for _, r := range line {
				if r < '0' || r > '9' {
					isDigits = false
					break
				}
			}
			if isDigits {
				continue
			}
		}

		result.TotalRaw++

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
		if length < config.MinLength || length > config.MaxLength {
			continue
		}

		wordType := fmt.Sprintf("%d-c", length)
		ln := lineNum

		source := schema.WordSource{
			DictName:     dictName,
			DictFilepath: absPath,
			Language:     config.Language,
			OriginalForm: word,
			LineNumber:   &ln,
			Category:     config.Category,
		}

		if existing, ok := words[normalized]; ok {
			existing.AddSource(source)
			result.TotalDuplicates++
		} else {
			w := schema.NewWord(normalized, length, wordType)
			w.AddSource(source)
			words[normalized] = w
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	for _, w := range words {
		result.Words = append(result.Words, w)
	}
	result.TotalValid = len(words)

	return result, nil
}

// DownloadAndIngest downloads and ingests a Hunspell dictionary.
func DownloadAndIngest(language, cacheDir string, config IngestConfig, force bool) (*IngestResult, error) {
	path, err := Download(language, cacheDir, force)
	if err != nil {
		return nil, err
	}
	return IngestHunspell(path, config)
}

// GetSupportedLanguages returns list of supported language codes.
func GetSupportedLanguages() []string {
	langs := make([]string, 0, len(HunspellURLs))
	for lang := range HunspellURLs {
		langs = append(langs, lang)
	}
	return langs
}
