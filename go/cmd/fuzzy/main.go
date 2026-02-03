// ditong-fuzzy - Fuzzy word search using BK-tree.
// Usage: ditong-fuzzy [options] <query>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ditong/internal/similarity"

	"github.com/spf13/pflag"
)

func main() {
	// Flags
	dictDir := pflag.StringP("dict-dir", "d", "", "Directory containing dictionary JSON files")
	maxDistance := pflag.IntP("distance", "n", 2, "Maximum edit distance")
	limit := pflag.IntP("limit", "l", 10, "Maximum results to show")
	jsonOutput := pflag.BoolP("json", "j", false, "Output as JSON")
	language := pflag.StringP("language", "L", "", "Filter by language (empty = all)")
	wordType := pflag.StringP("type", "t", "", "Filter by word type (e.g., '5-c')")

	pflag.Parse()

	if pflag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ditong-fuzzy [options] <query>")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	query := strings.ToLower(pflag.Arg(0))

	// Find dictionary directory
	if *dictDir == "" {
		// Try common locations
		candidates := []string{
			"dicts",
			"../dicts",
			"../../dicts",
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				*dictDir = c
				break
			}
		}
		if *dictDir == "" {
			fmt.Fprintln(os.Stderr, "Error: dictionary directory not found. Use --dict-dir")
			os.Exit(1)
		}
	}

	// Load words from dictionaries
	words := loadWords(*dictDir, *language, *wordType)
	if len(words) == 0 {
		fmt.Fprintln(os.Stderr, "No words found in dictionary")
		os.Exit(1)
	}

	// Build BK-tree
	tree := similarity.NewBKTree()
	tree.InsertAll(words)

	// Search
	results := tree.Search(query, *maxDistance)

	// Sort by distance, then alphabetically
	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance != results[j].Distance {
			return results[i].Distance < results[j].Distance
		}
		return results[i].Word < results[j].Word
	})

	// Limit results
	if *limit > 0 && len(results) > *limit {
		results = results[:*limit]
	}

	// Output
	if *jsonOutput {
		output := struct {
			Query    string                    `json:"query"`
			MaxDist  int                       `json:"max_distance"`
			Count    int                       `json:"count"`
			Results  []similarity.SearchResult `json:"results"`
		}{
			Query:   query,
			MaxDist: *maxDistance,
			Count:   len(results),
			Results: results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
	} else {
		if len(results) == 0 {
			fmt.Printf("No matches found for %q within distance %d\n", query, *maxDistance)
			return
		}

		fmt.Printf("Fuzzy matches for %q (max distance: %d):\n\n", query, *maxDistance)
		for _, r := range results {
			fmt.Printf("  %s (distance: %d)\n", r.Word, r.Distance)
		}
		fmt.Printf("\n%d result(s) found\n", len(results))
	}
}

// loadWords loads words from dictionary JSON files.
func loadWords(dictDir, language, wordType string) []string {
	var words []string

	err := filepath.Walk(dictDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Skip config files
		if strings.HasPrefix(info.Name(), "_") {
			return nil
		}

		// Filter by word type if specified
		if wordType != "" && !strings.Contains(info.Name(), wordType) {
			return nil
		}

		// Filter by language if specified
		if language != "" {
			rel, _ := filepath.Rel(dictDir, path)
			if !strings.HasPrefix(rel, language+string(filepath.Separator)) &&
				!strings.HasPrefix(rel, "synthesis") {
				return nil
			}
		}

		// Parse dictionary
		dictWords := parseDictionary(path)
		words = append(words, dictWords...)

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error walking directory: %v\n", err)
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(words))
	for _, w := range words {
		if !seen[w] {
			seen[w] = true
			unique = append(unique, w)
		}
	}

	return unique
}

// parseDictionary extracts words from a dictionary JSON file.
func parseDictionary(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Try format 1: words as map {"words": {"word1": {...}, "word2": {...}}}
	var dictMap struct {
		Words map[string]struct {
			Normalized string `json:"normalized"`
		} `json:"words"`
	}
	if err := json.Unmarshal(data, &dictMap); err == nil && len(dictMap.Words) > 0 {
		words := make([]string, 0, len(dictMap.Words))
		for _, w := range dictMap.Words {
			if w.Normalized != "" {
				words = append(words, w.Normalized)
			}
		}
		return words
	}

	// Try format 2: words as array {"words": [{"normalized": "..."}, ...]}
	var dictArray struct {
		Words []struct {
			Normalized string `json:"normalized"`
		} `json:"words"`
	}
	if err := json.Unmarshal(data, &dictArray); err == nil && len(dictArray.Words) > 0 {
		words := make([]string, 0, len(dictArray.Words))
		for _, w := range dictArray.Words {
			if w.Normalized != "" {
				words = append(words, w.Normalized)
			}
		}
		return words
	}

	// Try format 3: simple string array {"words": ["word1", "word2"]}
	var dictSimple struct {
		Words []string `json:"words"`
	}
	if err := json.Unmarshal(data, &dictSimple); err == nil {
		return dictSimple.Words
	}

	return nil
}
