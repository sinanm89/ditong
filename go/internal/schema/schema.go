// Package schema defines word and dictionary data structures for ditong.
package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// WordSource tracks where a word came from.
type WordSource struct {
	DictName     string `json:"dict_name"`
	DictFilepath string `json:"dict_filepath"`
	Language     string `json:"language"`
	OriginalForm string `json:"original_form"`
	LineNumber   *int   `json:"line_number,omitempty"`
	Category     string `json:"category"`
}

// Word represents a normalized word with full metadata.
type Word struct {
	Normalized      string            `json:"normalized"`
	Length          int               `json:"length"`
	WordType        string            `json:"type"`
	Sources         []WordSource      `json:"sources"`
	Categories      map[string]bool   `json:"-"` // Internal set
	Languages       map[string]bool   `json:"-"` // Internal set
	Tags            map[string]bool   `json:"-"` // Internal set
	IPA             string            `json:"ipa,omitempty"`
	SynthesisGroups map[string]bool   `json:"-"` // Internal set
}

// NewWord creates a new Word with initialized sets.
func NewWord(normalized string, length int, wordType string) *Word {
	return &Word{
		Normalized:      normalized,
		Length:          length,
		WordType:        wordType,
		Sources:         []WordSource{},
		Categories:      make(map[string]bool),
		Languages:       make(map[string]bool),
		Tags:            make(map[string]bool),
		SynthesisGroups: make(map[string]bool),
	}
}

// AddSource adds a source and updates derived fields.
func (w *Word) AddSource(source WordSource) {
	w.Sources = append(w.Sources, source)
	w.Categories[source.Category] = true
	w.Languages[source.Language] = true
}

// GetSourceDicts returns list of source dictionary names.
func (w *Word) GetSourceDicts() []string {
	dicts := make([]string, len(w.Sources))
	for i, s := range w.Sources {
		dicts[i] = s.DictName
	}
	return dicts
}

// MatchesFilter checks if word matches filter criteria.
func (w *Word) MatchesFilter(
	includeCategories, excludeCategories map[string]bool,
	includeLanguages map[string]bool,
	minLength, maxLength int,
) bool {
	if minLength > 0 && w.Length < minLength {
		return false
	}
	if maxLength > 0 && w.Length > maxLength {
		return false
	}

	if len(includeLanguages) > 0 {
		found := false
		for lang := range w.Languages {
			if includeLanguages[lang] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(includeCategories) > 0 {
		found := false
		for cat := range w.Categories {
			if includeCategories[cat] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(excludeCategories) > 0 {
		for cat := range w.Categories {
			if excludeCategories[cat] {
				return false
			}
		}
	}

	return true
}

// MarshalJSON implements custom JSON marshaling.
func (w *Word) MarshalJSON() ([]byte, error) {
	type Alias Word

	categories := make([]string, 0, len(w.Categories))
	for k := range w.Categories {
		categories = append(categories, k)
	}
	sort.Strings(categories)

	languages := make([]string, 0, len(w.Languages))
	for k := range w.Languages {
		languages = append(languages, k)
	}
	sort.Strings(languages)

	tags := make([]string, 0, len(w.Tags))
	for k := range w.Tags {
		tags = append(tags, k)
	}
	sort.Strings(tags)

	synthGroups := make([]string, 0, len(w.SynthesisGroups))
	for k := range w.SynthesisGroups {
		synthGroups = append(synthGroups, k)
	}
	sort.Strings(synthGroups)

	return json.Marshal(&struct {
		*Alias
		Categories      []string `json:"categories"`
		Languages       []string `json:"languages"`
		Tags            []string `json:"tags"`
		SynthesisGroups []string `json:"synthesis_groups"`
	}{
		Alias:           (*Alias)(w),
		Categories:      categories,
		Languages:       languages,
		Tags:            tags,
		SynthesisGroups: synthGroups,
	})
}

// Dictionary is a collection of words with metadata.
type Dictionary struct {
	Name        string           `json:"name"`
	Language    string           `json:"language,omitempty"`
	Languages   map[string]bool  `json:"-"`
	Words       map[string]*Word `json:"-"` // normalized -> Word
	GeneratedAt string           `json:"generated_at"`
	SourceDicts map[string]bool  `json:"-"`
	WordType    string           `json:"word_type,omitempty"`
}

// NewDictionary creates a new Dictionary.
func NewDictionary(name string) *Dictionary {
	return &Dictionary{
		Name:        name,
		Languages:   make(map[string]bool),
		Words:       make(map[string]*Word),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SourceDicts: make(map[string]bool),
	}
}

// AddWord adds or merges a word.
func (d *Dictionary) AddWord(word *Word) {
	if existing, ok := d.Words[word.Normalized]; ok {
		for _, source := range word.Sources {
			existing.AddSource(source)
		}
		for tag := range word.Tags {
			existing.Tags[tag] = true
		}
	} else {
		d.Words[word.Normalized] = word
	}

	for lang := range word.Languages {
		d.Languages[lang] = true
	}
	for _, dict := range word.GetSourceDicts() {
		d.SourceDicts[dict] = true
	}
}

// Count returns word count.
func (d *Dictionary) Count() int {
	return len(d.Words)
}

// GetWordsSorted returns sorted list of words.
func (d *Dictionary) GetWordsSorted() []*Word {
	words := make([]*Word, 0, len(d.Words))
	for _, w := range d.Words {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool {
		return words[i].Normalized < words[j].Normalized
	})
	return words
}

// MarshalJSON implements custom JSON marshaling.
func (d *Dictionary) MarshalJSON() ([]byte, error) {
	languages := make([]string, 0, len(d.Languages))
	for k := range d.Languages {
		languages = append(languages, k)
	}
	sort.Strings(languages)

	sourceDicts := make([]string, 0, len(d.SourceDicts))
	for k := range d.SourceDicts {
		sourceDicts = append(sourceDicts, k)
	}
	sort.Strings(sourceDicts)

	wordsMap := make(map[string]*Word)
	for _, w := range d.GetWordsSorted() {
		wordsMap[w.Normalized] = w
	}

	return json.Marshal(&struct {
		Name        string           `json:"name"`
		Language    string           `json:"language,omitempty"`
		Languages   []string         `json:"languages"`
		WordType    string           `json:"word_type,omitempty"`
		GeneratedAt string           `json:"generated_at"`
		SourceDicts []string         `json:"source_dicts"`
		WordCount   int              `json:"word_count"`
		Words       map[string]*Word `json:"words"`
	}{
		Name:        d.Name,
		Language:    d.Language,
		Languages:   languages,
		WordType:    d.WordType,
		GeneratedAt: d.GeneratedAt,
		SourceDicts: sourceDicts,
		WordCount:   d.Count(),
		Words:       wordsMap,
	})
}

// Save saves dictionary to JSON file.
func (d *Dictionary) Save(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(d)
}

// WordTypeFromLength returns word type string from length.
func WordTypeFromLength(length int) string {
	if length >= 3 && length <= 10 {
		return string(rune('0'+length/10)) + string(rune('0'+length%10)) + "-c"[1:]
	}
	return ""
}
