package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWord(t *testing.T) {
	word := NewWord("hello", 5, "5-c")

	if word.Normalized != "hello" {
		t.Errorf("Normalized = %q, want %q", word.Normalized, "hello")
	}
	if word.Length != 5 {
		t.Errorf("Length = %d, want %d", word.Length, 5)
	}
	if word.WordType != "5-c" {
		t.Errorf("WordType = %q, want %q", word.WordType, "5-c")
	}
	if len(word.Sources) != 0 {
		t.Errorf("Sources should be empty, got %d", len(word.Sources))
	}
}

func TestWordAddSource(t *testing.T) {
	word := NewWord("care", 4, "4-c")

	sourceEN := WordSource{
		DictName:     "hunspell_en",
		DictFilepath: "/en.dic",
		Language:     "en",
		OriginalForm: "care",
		Category:     "standard",
	}

	sourceTR := WordSource{
		DictName:     "hunspell_tr",
		DictFilepath: "/tr.dic",
		Language:     "tr",
		OriginalForm: "çare",
		Category:     "standard",
	}

	word.AddSource(sourceEN)
	word.AddSource(sourceTR)

	if len(word.Sources) != 2 {
		t.Errorf("Sources count = %d, want 2", len(word.Sources))
	}
	if !word.Languages["en"] {
		t.Error("Languages should contain 'en'")
	}
	if !word.Languages["tr"] {
		t.Error("Languages should contain 'tr'")
	}
	if !word.Categories["standard"] {
		t.Error("Categories should contain 'standard'")
	}
}

func TestWordGetSourceDicts(t *testing.T) {
	word := NewWord("test", 4, "4-c")
	word.AddSource(WordSource{
		DictName:     "dict1",
		DictFilepath: "/d1",
		Language:     "en",
		OriginalForm: "test",
	})
	word.AddSource(WordSource{
		DictName:     "dict2",
		DictFilepath: "/d2",
		Language:     "tr",
		OriginalForm: "test",
	})

	dicts := word.GetSourceDicts()
	if len(dicts) != 2 {
		t.Errorf("GetSourceDicts count = %d, want 2", len(dicts))
	}

	found1, found2 := false, false
	for _, d := range dicts {
		if d == "dict1" {
			found1 = true
		}
		if d == "dict2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("GetSourceDicts should contain both dict1 and dict2")
	}
}

func TestWordMatchesFilter(t *testing.T) {
	word := NewWord("hello", 5, "5-c")
	word.AddSource(WordSource{
		DictName:     "test",
		DictFilepath: "/test",
		Language:     "en",
		OriginalForm: "hello",
		Category:     "standard",
	})

	tests := []struct {
		name              string
		includeCategories map[string]bool
		excludeCategories map[string]bool
		includeLanguages  map[string]bool
		minLength         int
		maxLength         int
		expected          bool
	}{
		{
			name:      "matches all",
			minLength: 3, maxLength: 8,
			expected: true,
		},
		{
			name:      "fails min length",
			minLength: 6, maxLength: 10,
			expected: false,
		},
		{
			name:      "fails max length",
			minLength: 1, maxLength: 4,
			expected: false,
		},
		{
			name:             "matches language",
			includeLanguages: map[string]bool{"en": true},
			expected:         true,
		},
		{
			name:             "fails language",
			includeLanguages: map[string]bool{"tr": true},
			expected:         false,
		},
		{
			name:              "matches category",
			includeCategories: map[string]bool{"standard": true},
			expected:          true,
		},
		{
			name:              "fails include category",
			includeCategories: map[string]bool{"urban": true},
			expected:          false,
		},
		{
			name:              "passes exclude category",
			excludeCategories: map[string]bool{"urban": true},
			expected:          true,
		},
		{
			name:              "fails exclude category",
			excludeCategories: map[string]bool{"standard": true},
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := word.MatchesFilter(
				tt.includeCategories,
				tt.excludeCategories,
				tt.includeLanguages,
				tt.minLength,
				tt.maxLength,
			)
			if result != tt.expected {
				t.Errorf("MatchesFilter = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWordMarshalJSON(t *testing.T) {
	word := NewWord("hello", 5, "5-c")
	word.AddSource(WordSource{
		DictName:     "test",
		DictFilepath: "/test",
		Language:     "en",
		OriginalForm: "hello",
		Category:     "standard",
	})
	word.Tags["tag1"] = true

	data, err := json.Marshal(word)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["normalized"] != "hello" {
		t.Errorf("normalized = %v, want hello", result["normalized"])
	}
	if int(result["length"].(float64)) != 5 {
		t.Errorf("length = %v, want 5", result["length"])
	}
}

func TestNewDictionary(t *testing.T) {
	d := NewDictionary("test_dict")

	if d.Name != "test_dict" {
		t.Errorf("Name = %q, want %q", d.Name, "test_dict")
	}
	if d.Count() != 0 {
		t.Errorf("Count = %d, want 0", d.Count())
	}
}

func TestDictionaryAddWord(t *testing.T) {
	d := NewDictionary("test")

	word := NewWord("hello", 5, "5-c")
	word.AddSource(WordSource{
		DictName:     "test",
		DictFilepath: "/test",
		Language:     "en",
		OriginalForm: "hello",
	})

	d.AddWord(word)

	if d.Count() != 1 {
		t.Errorf("Count = %d, want 1", d.Count())
	}
	if _, ok := d.Words["hello"]; !ok {
		t.Error("Dictionary should contain 'hello'")
	}
}

func TestDictionaryAddWordMerge(t *testing.T) {
	d := NewDictionary("test")

	word1 := NewWord("care", 4, "4-c")
	word1.AddSource(WordSource{
		DictName:     "en",
		DictFilepath: "/en",
		Language:     "en",
		OriginalForm: "care",
	})

	word2 := NewWord("care", 4, "4-c")
	word2.AddSource(WordSource{
		DictName:     "tr",
		DictFilepath: "/tr",
		Language:     "tr",
		OriginalForm: "çare",
	})

	d.AddWord(word1)
	d.AddWord(word2)

	if d.Count() != 1 {
		t.Errorf("Count = %d, want 1 (should merge)", d.Count())
	}
	if len(d.Words["care"].Sources) != 2 {
		t.Errorf("Sources count = %d, want 2", len(d.Words["care"].Sources))
	}
}

func TestDictionaryGetWordsSorted(t *testing.T) {
	d := NewDictionary("test")
	d.AddWord(NewWord("zebra", 5, "5-c"))
	d.AddWord(NewWord("apple", 5, "5-c"))
	d.AddWord(NewWord("mango", 5, "5-c"))

	words := d.GetWordsSorted()

	if len(words) != 3 {
		t.Fatalf("GetWordsSorted count = %d, want 3", len(words))
	}
	if words[0].Normalized != "apple" {
		t.Errorf("First word = %q, want apple", words[0].Normalized)
	}
	if words[1].Normalized != "mango" {
		t.Errorf("Second word = %q, want mango", words[1].Normalized)
	}
	if words[2].Normalized != "zebra" {
		t.Errorf("Third word = %q, want zebra", words[2].Normalized)
	}
}

func TestDictionarySave(t *testing.T) {
	d := NewDictionary("test_save")
	d.Language = "en"
	d.WordType = "5-c"

	word := NewWord("hello", 5, "5-c")
	word.AddSource(WordSource{
		DictName:     "test",
		DictFilepath: "/test",
		Language:     "en",
		OriginalForm: "hello",
	})
	d.AddWord(word)

	tmpDir := t.TempDir()
	filepath := filepath.Join(tmpDir, "test.json")

	if err := d.Save(filepath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Error("Save did not create file")
	}

	// Verify content
	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["name"] != "test_save" {
		t.Errorf("name = %v, want test_save", result["name"])
	}
	if int(result["word_count"].(float64)) != 1 {
		t.Errorf("word_count = %v, want 1", result["word_count"])
	}
}

func TestWordTypeFromLength(t *testing.T) {
	tests := []struct {
		length   int
		expected string
	}{
		{3, "3-c"},
		{4, "4-c"},
		{5, "5-c"},
		{10, "10-c"},
		{2, ""},
		{11, ""},
	}

	for _, tt := range tests {
		result := WordTypeFromLength(tt.length)
		if result != tt.expected {
			t.Errorf("WordTypeFromLength(%d) = %q, want %q", tt.length, result, tt.expected)
		}
	}
}
