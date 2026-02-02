// Package similarity provides similarity search using BK-trees.
package similarity

// BKTree is a BK-tree for similarity search using edit distance.
// BK-trees are space-partitioning data structures for metric spaces,
// particularly useful for spelling correction and fuzzy matching.
type BKTree struct {
	root *bkNode
	size int
}

// bkNode represents a node in the BK-tree.
type bkNode struct {
	word     string
	children map[int]*bkNode
}

// NewBKTree creates a new empty BK-tree.
func NewBKTree() *BKTree {
	return &BKTree{}
}

// Insert adds a word to the tree.
func (t *BKTree) Insert(word string) {
	if word == "" {
		return
	}

	if t.root == nil {
		t.root = &bkNode{
			word:     word,
			children: make(map[int]*bkNode),
		}
		t.size++
		return
	}

	current := t.root
	for {
		dist := LevenshteinDistance(word, current.word)
		if dist == 0 {
			return // Word already exists
		}

		child, exists := current.children[dist]
		if !exists {
			current.children[dist] = &bkNode{
				word:     word,
				children: make(map[int]*bkNode),
			}
			t.size++
			return
		}
		current = child
	}
}

// InsertAll adds multiple words to the tree.
func (t *BKTree) InsertAll(words []string) {
	for _, word := range words {
		t.Insert(word)
	}
}

// SearchResult holds a search result with its distance.
type SearchResult struct {
	Word     string
	Distance int
}

// Search finds all words within maxDistance edit distance from the query.
func (t *BKTree) Search(query string, maxDistance int) []SearchResult {
	if t.root == nil || query == "" {
		return nil
	}

	var results []SearchResult
	t.searchNode(t.root, query, maxDistance, &results)
	return results
}

// searchNode recursively searches the tree.
func (t *BKTree) searchNode(node *bkNode, query string, maxDistance int, results *[]SearchResult) {
	dist := LevenshteinDistance(query, node.word)

	if dist <= maxDistance {
		*results = append(*results, SearchResult{
			Word:     node.word,
			Distance: dist,
		})
	}

	// Only search children within the possible distance range
	minDist := dist - maxDistance
	maxDist := dist + maxDistance

	for childDist, child := range node.children {
		if childDist >= minDist && childDist <= maxDist {
			t.searchNode(child, query, maxDistance, results)
		}
	}
}

// Size returns the number of words in the tree.
func (t *BKTree) Size() int {
	return t.size
}

// Contains checks if a word exists in the tree.
func (t *BKTree) Contains(word string) bool {
	results := t.Search(word, 0)
	return len(results) > 0 && results[0].Distance == 0
}

// LevenshteinDistance calculates the edit distance between two strings.
// This is an optimized implementation using only two rows of the matrix.
func LevenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	r1 := []rune(s1)
	r2 := []rune(s2)

	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Ensure s1 is the shorter string for space optimization
	if len1 > len2 {
		r1, r2 = r2, r1
		len1, len2 = len2, len1
	}

	// Use two rows instead of full matrix
	prev := make([]int, len1+1)
	curr := make([]int, len1+1)

	// Initialize first row
	for i := 0; i <= len1; i++ {
		prev[i] = i
	}

	// Fill the matrix
	for j := 1; j <= len2; j++ {
		curr[0] = j

		for i := 1; i <= len1; i++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			curr[i] = min(
				prev[i]+1,      // deletion
				curr[i-1]+1,    // insertion
				prev[i-1]+cost, // substitution
			)
		}

		prev, curr = curr, prev
	}

	return prev[len1]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
