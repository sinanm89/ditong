package similarity

import (
	"sort"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1, s2   string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"Saturday", "Sunday", 3},
		{"hello", "hallo", 1},
		{"book", "back", 2},
		{"test", "tests", 1},
		{"flaw", "lawn", 2},
	}

	for _, tt := range tests {
		result := LevenshteinDistance(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d",
				tt.s1, tt.s2, result, tt.expected)
		}

		// Test symmetry
		reverse := LevenshteinDistance(tt.s2, tt.s1)
		if reverse != result {
			t.Errorf("LevenshteinDistance is not symmetric: (%q, %q) = %d, (%q, %q) = %d",
				tt.s1, tt.s2, result, tt.s2, tt.s1, reverse)
		}
	}
}

func TestBKTreeInsert(t *testing.T) {
	tree := NewBKTree()

	words := []string{"hello", "hallo", "help", "world", "word"}
	tree.InsertAll(words)

	if tree.Size() != 5 {
		t.Errorf("Size() = %d, want 5", tree.Size())
	}

	// Test duplicate insertion
	tree.Insert("hello")
	if tree.Size() != 5 {
		t.Errorf("Size() after duplicate = %d, want 5", tree.Size())
	}
}

func TestBKTreeContains(t *testing.T) {
	tree := NewBKTree()
	tree.InsertAll([]string{"hello", "world", "test"})

	if !tree.Contains("hello") {
		t.Error("Contains(hello) = false, want true")
	}

	if tree.Contains("xyz") {
		t.Error("Contains(xyz) = true, want false")
	}
}

func TestBKTreeSearch(t *testing.T) {
	tree := NewBKTree()
	words := []string{"hello", "hallo", "help", "held", "hero", "world", "word", "work"}
	tree.InsertAll(words)

	tests := []struct {
		query    string
		maxDist  int
		expected []string
	}{
		{"hello", 0, []string{"hello"}},
		{"hello", 1, []string{"hello", "hallo"}},
		{"hello", 2, []string{"hello", "hallo", "help", "held", "hero"}},
		{"world", 1, []string{"world", "word"}}, // "work" is distance 2
		{"xyz", 10, words}, // Large distance should match all
	}

	for _, tt := range tests {
		results := tree.Search(tt.query, tt.maxDist)
		resultWords := make([]string, len(results))
		for i, r := range results {
			resultWords[i] = r.Word
		}

		sort.Strings(resultWords)
		sort.Strings(tt.expected)

		if len(resultWords) != len(tt.expected) {
			t.Errorf("Search(%q, %d) returned %d results, want %d: got %v, want %v",
				tt.query, tt.maxDist, len(resultWords), len(tt.expected),
				resultWords, tt.expected)
			continue
		}

		for i, word := range resultWords {
			if word != tt.expected[i] {
				t.Errorf("Search(%q, %d) mismatch at %d: got %q, want %q",
					tt.query, tt.maxDist, i, word, tt.expected[i])
			}
		}
	}
}

func TestBKTreeSearchEmpty(t *testing.T) {
	tree := NewBKTree()

	results := tree.Search("hello", 1)
	if results != nil && len(results) > 0 {
		t.Errorf("Search on empty tree returned %v, want nil", results)
	}

	results = tree.Search("", 1)
	if results != nil && len(results) > 0 {
		t.Errorf("Search with empty query returned %v, want nil", results)
	}
}

func TestBKTreeSearchDistance(t *testing.T) {
	tree := NewBKTree()
	tree.InsertAll([]string{"book", "cook", "look", "took", "back"})

	results := tree.Search("book", 1)
	for _, r := range results {
		if r.Distance > 1 {
			t.Errorf("Result %q has distance %d, want <= 1", r.Word, r.Distance)
		}
		actualDist := LevenshteinDistance("book", r.Word)
		if actualDist != r.Distance {
			t.Errorf("Reported distance %d != actual distance %d for %q",
				r.Distance, actualDist, r.Word)
		}
	}
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	s1 := "kitten"
	s2 := "sitting"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LevenshteinDistance(s1, s2)
	}
}

func BenchmarkBKTreeInsert(b *testing.B) {
	words := generateWords(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree := NewBKTree()
		tree.InsertAll(words)
	}
}

func BenchmarkBKTreeSearch(b *testing.B) {
	tree := NewBKTree()
	tree.InsertAll(generateWords(10000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Search("hello", 2)
	}
}

// generateWords creates a list of test words.
func generateWords(n int) []string {
	base := []string{"hello", "world", "test", "word", "book", "look", "cook", "help"}
	words := make([]string, n)
	for i := 0; i < n; i++ {
		words[i] = base[i%len(base)] + string(rune('a'+(i%26)))
	}
	return words
}
