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

// CursewordURLs maps language codes to curseword list URLs.
// Sources: Various open-source profanity lists.
var CursewordURLs = map[string]string{
	"en": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/en",
	"tr": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/tr",
	"de": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/de",
	"fr": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/fr",
	"es": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/es",
	"it": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/it",
	"pt": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/pt",
	"nl": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/nl",
	"pl": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/pl",
	"ru": "https://raw.githubusercontent.com/LDNOOBW/List-of-Dirty-Naughty-Obscene-and-Otherwise-Bad-Words/master/ru",
}

// CursewordConfig returns config for curseword ingestion.
func CursewordConfig(language string) IngestConfig {
	return IngestConfig{
		Language:  language,
		Category:  "curseword",
		MinLength: 3,
		MaxLength: 15, // Cursewords can be longer compound words
	}
}

// DownloadCursewords downloads a curseword list to the cache directory.
func DownloadCursewords(language, cacheDir string, force bool) (string, error) {
	url, ok := CursewordURLs[language]
	if !ok {
		return "", fmt.Errorf("no curseword list for language: %s", language)
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	filename := fmt.Sprintf("%s_cursewords.txt", language)
	cachedPath := filepath.Join(cacheDir, filename)

	if !force {
		if _, err := os.Stat(cachedPath); err == nil {
			return cachedPath, nil
		}
	}

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

	return cachedPath, nil
}

// IngestCursewords ingests a plain text curseword list.
func IngestCursewords(filePath string, config IngestConfig) (*IngestResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	absPath, _ := filepath.Abs(filePath)
	dictName := fmt.Sprintf("cursewords_%s", config.Language)

	result := &IngestResult{
		SourcePath: absPath,
		DictName:   dictName,
		Language:   config.Language,
		Category:   "curseword",
	}

	words := make(map[string]*schema.Word)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		result.TotalRaw++

		normalized := normalizer.NormalizeAndValidate(line)
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
			OriginalForm: line,
			LineNumber:   &ln,
			Category:     "curseword",
		}

		if existing, ok := words[normalized]; ok {
			existing.AddSource(source)
			result.TotalDuplicates++
		} else {
			w := schema.NewWord(normalized, length, wordType)
			w.AddSource(source)
			w.Tags["curseword"] = true
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

// DownloadAndIngestCursewords downloads and ingests a curseword list.
func DownloadAndIngestCursewords(language, cacheDir string, config IngestConfig, force bool) (*IngestResult, error) {
	path, err := DownloadCursewords(language, cacheDir, force)
	if err != nil {
		return nil, err
	}
	return IngestCursewords(path, config)
}

// GetCursewordLanguages returns list of languages with curseword support.
func GetCursewordLanguages() []string {
	langs := make([]string, 0, len(CursewordURLs))
	for lang := range CursewordURLs {
		langs = append(langs, lang)
	}
	return langs
}

// HasCursewordSupport checks if a language has curseword list available.
func HasCursewordSupport(language string) bool {
	_, ok := CursewordURLs[language]
	return ok
}
