// ditong-fuzzy - Interactive fuzzy word search using BK-tree.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"ditong/internal/similarity"

	"github.com/pterm/pterm"
	"github.com/spf13/pflag"
)

var tree *similarity.BKTree
var wordCount int

func main() {
	// Flags
	dictDir := pflag.StringP("dict-dir", "d", "", "Directory containing dictionary JSON files")
	maxDistance := pflag.IntP("distance", "n", 2, "Maximum edit distance")
	limit := pflag.IntP("limit", "l", 10, "Maximum results to show")
	jsonOutput := pflag.BoolP("json", "j", false, "Output as JSON")
	language := pflag.StringP("language", "L", "", "Filter by language (empty = all)")
	wordType := pflag.StringP("type", "t", "", "Filter by word type (e.g., '5-c')")
	interactive := pflag.BoolP("interactive", "i", false, "Run in interactive mode")

	pflag.Parse()

	// If no args and no interactive flag, default to interactive
	if pflag.NArg() < 1 && !*jsonOutput {
		*interactive = true
	}

	// Find dictionary directory
	if *dictDir == "" {
		candidates := []string{
			"output/dicts",
			"output/consolidated",
			"output",
			"dicts",
			"../output/dicts",
			"../output",
			"../dicts",
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				*dictDir = c
				break
			}
		}
	}

	if *interactive {
		runInteractive(*dictDir, *maxDistance, *limit, *language, *wordType)
		return
	}

	// Non-interactive mode
	if pflag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ditong-fuzzy [options] <query>")
		fmt.Fprintln(os.Stderr, "       ditong-fuzzy -i  (interactive mode)")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *dictDir == "" {
		fmt.Fprintln(os.Stderr, "Error: dictionary directory not found. Use --dict-dir")
		os.Exit(1)
	}

	query := strings.ToLower(pflag.Arg(0))
	words := loadWords(*dictDir, *language, *wordType)
	if len(words) == 0 {
		fmt.Fprintln(os.Stderr, "No words found in dictionary")
		os.Exit(1)
	}

	tree := similarity.NewBKTree()
	tree.InsertAll(words)

	results := search(tree, query, *maxDistance, *limit)
	outputResults(results, query, *maxDistance, *jsonOutput)
}

func runInteractive(dictDir string, defaultDistance, defaultLimit int, language, wordType string) {
	// Header
	fmt.Println()
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("fuzzy", pterm.NewStyle(pterm.FgCyan)),
	).Render()

	pterm.DefaultBox.WithTitle(pterm.FgCyan.Sprint("Fuzzy Search")).
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightBlue)).
		Println(
			pterm.FgWhite.Sprint("BK-tree similarity search") + "\n" +
				pterm.FgGray.Sprint("Type a word to find similar matches. Commands: :q quit, :d N set distance, :l N set limit"),
		)
	fmt.Println()

	// Get dictionary directory
	if dictDir == "" {
		dictDir, _ = pterm.DefaultInteractiveTextInput.
			WithDefaultValue("./output/dicts").
			Show("Dictionary directory")
	}

	if _, err := os.Stat(dictDir); os.IsNotExist(err) {
		pterm.Error.Println("Directory not found:", dictDir)
		os.Exit(1)
	}

	// Load words with spinner
	spinner, _ := pterm.DefaultSpinner.Start("Loading dictionary...")
	words := loadWords(dictDir, language, wordType)
	if len(words) == 0 {
		spinner.Fail("No words found in dictionary")
		os.Exit(1)
	}

	spinner.UpdateText("Building search index...")
	tree = similarity.NewBKTree()
	tree.InsertAll(words)
	wordCount = len(words)
	spinner.Success(fmt.Sprintf("Loaded %s words", pterm.FgCyan.Sprintf("%d", wordCount)))
	fmt.Println()

	distance := defaultDistance
	limit := defaultLimit

	// Search loop
	for {
		query, _ := pterm.DefaultInteractiveTextInput.
			WithDefaultText(pterm.FgGray.Sprint("(distance: "+strconv.Itoa(distance)+")")).
			Show(pterm.FgCyan.Sprint("Search"))

		query = strings.TrimSpace(strings.ToLower(query))
		if query == "" {
			continue
		}

		// Commands
		if strings.HasPrefix(query, ":") {
			if query == ":q" || query == ":quit" || query == ":exit" {
				pterm.Info.Println("Goodbye!")
				return
			}
			if strings.HasPrefix(query, ":d ") {
				if n, err := strconv.Atoi(strings.TrimPrefix(query, ":d ")); err == nil && n >= 0 {
					distance = n
					pterm.Info.Println("Distance set to", pterm.FgCyan.Sprint(distance))
				}
				continue
			}
			if strings.HasPrefix(query, ":l ") {
				if n, err := strconv.Atoi(strings.TrimPrefix(query, ":l ")); err == nil && n > 0 {
					limit = n
					pterm.Info.Println("Limit set to", pterm.FgCyan.Sprint(limit))
				}
				continue
			}
			pterm.Warning.Println("Unknown command. Use :q to quit, :d N for distance, :l N for limit")
			continue
		}

		// Search
		results := search(tree, query, distance, limit)

		if len(results) == 0 {
			pterm.Warning.Printfln("No matches for %q within distance %d", query, distance)
			fmt.Println()
			continue
		}

		// Display results as table
		tableData := pterm.TableData{{"Word", "Distance"}}
		for _, r := range results {
			distStr := strconv.Itoa(r.Distance)
			if r.Distance == 0 {
				distStr = pterm.FgGreen.Sprint("0 (exact)")
			}
			tableData = append(tableData, []string{r.Word, distStr})
		}

		pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()
		pterm.Info.Printfln("Found %d matches for %q", len(results), query)

		// Show related shorter/longer words
		showRelatedWords(tree, query, distance)
		fmt.Println()
	}
}

func showRelatedWords(tree *similarity.BKTree, query string, distance int) {
	qLen := len(query)
	var related []string

	// Show shorter versions (prefixes that exist as words)
	for i := qLen - 1; i >= 3; i-- {
		prefix := query[:i]
		if tree.Contains(prefix) {
			related = append(related, fmt.Sprintf("%s (%d chars)", prefix, i))
		}
	}

	// Show longer versions (query as prefix of other words)
	if qLen >= 3 {
		longer := tree.Search(query, distance)
		for _, r := range longer {
			if len(r.Word) > qLen && strings.HasPrefix(r.Word, query) {
				related = append(related, fmt.Sprintf("%s (%d chars)", r.Word, len(r.Word)))
				if len(related) > 5 {
					break
				}
			}
		}
	}

	if len(related) > 0 {
		pterm.FgGray.Print("  Related: ")
		pterm.FgCyan.Println(strings.Join(related, ", "))
	}
}

func search(tree *similarity.BKTree, query string, maxDistance, limit int) []similarity.SearchResult {
	results := tree.Search(query, maxDistance)

	sort.Slice(results, func(i, j int) bool {
		if results[i].Distance != results[j].Distance {
			return results[i].Distance < results[j].Distance
		}
		return results[i].Word < results[j].Word
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

func outputResults(results []similarity.SearchResult, query string, maxDistance int, jsonOut bool) {
	if jsonOut {
		output := struct {
			Query   string                    `json:"query"`
			MaxDist int                       `json:"max_distance"`
			Count   int                       `json:"count"`
			Results []similarity.SearchResult `json:"results"`
		}{
			Query:   query,
			MaxDist: maxDistance,
			Count:   len(results),
			Results: results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(output)
		return
	}

	if len(results) == 0 {
		fmt.Printf("No matches found for %q within distance %d\n", query, maxDistance)
		return
	}

	fmt.Printf("Fuzzy matches for %q (max distance: %d):\n\n", query, maxDistance)
	for _, r := range results {
		fmt.Printf("  %s (distance: %d)\n", r.Word, r.Distance)
	}
	fmt.Printf("\n%d result(s) found\n", len(results))
}

func loadWords(dictDir, language, wordType string) []string {
	var words []string

	filepath.Walk(dictDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		if strings.HasPrefix(info.Name(), "_") {
			return nil
		}
		if wordType != "" && !strings.Contains(info.Name(), wordType) {
			return nil
		}
		if language != "" {
			rel, _ := filepath.Rel(dictDir, path)
			if !strings.HasPrefix(rel, language+string(filepath.Separator)) &&
				!strings.HasPrefix(rel, "synthesis") {
				return nil
			}
		}

		words = append(words, parseDictionary(path)...)
		return nil
	})

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

func parseDictionary(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Format 1: words as map
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

	// Format 2: words as object array
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

	// Format 3: simple string array
	var dictSimple struct {
		Words []string `json:"words"`
	}
	if err := json.Unmarshal(data, &dictSimple); err == nil {
		return dictSimple.Words
	}

	return nil
}
