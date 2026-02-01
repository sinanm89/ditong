// Package ui provides terminal UI components using pterm.
package ui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

// Theme colors for consistent styling
var (
	ColorPrimary   = pterm.FgCyan
	ColorSecondary = pterm.FgLightBlue
	ColorSuccess   = pterm.FgGreen
	ColorWarning   = pterm.FgYellow
	ColorError     = pterm.FgRed
	ColorMuted     = pterm.FgGray
)

// UI wraps pterm components for ditong.
type UI struct {
	quiet   bool
	verbose bool
}

// New creates a new UI instance.
func New(quiet, verbose bool) *UI {
	if quiet {
		pterm.DisableOutput()
	}
	return &UI{quiet: quiet, verbose: verbose}
}

// Banner prints the application banner.
func (u *UI) Banner() {
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("di", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle("tong", pterm.NewStyle(pterm.FgLightBlue)),
	).Render()

	pterm.DefaultCenter.Println(
		pterm.FgGray.Sprint("Multi-language Lexicon Toolkit"),
	)
	fmt.Println()
}

// Config prints the configuration summary.
func (u *UI) Config(languages []string, minLen, maxLen int, outputDir string) {
	pterm.DefaultSection.Println("Configuration")

	data := [][]string{
		{"Languages", fmt.Sprintf("%v", languages)},
		{"Length Range", fmt.Sprintf("%d - %d characters", minLen, maxLen)},
		{"Output", outputDir},
	}

	pterm.DefaultTable.WithData(data).Render()
	fmt.Println()
}

// Phase prints a phase header.
func (u *UI) Phase(number int, total int, name string) {
	pterm.DefaultSection.WithLevel(2).Println(
		fmt.Sprintf("[%d/%d] %s", number, total, name),
	)
}

// Spinner creates a spinner for long operations.
func (u *UI) Spinner(message string) *pterm.SpinnerPrinter {
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		Start(message)
	return spinner
}

// Progress creates a progress bar.
func (u *UI) Progress(title string, total int) *pterm.ProgressbarPrinter {
	pb, _ := pterm.DefaultProgressbar.
		WithTotal(total).
		WithTitle(title).
		WithShowElapsedTime(true).
		WithShowCount(true).
		Start()
	return pb
}

// LanguageStatus prints status for a language operation.
func (u *UI) LanguageStatus(lang string, status string, details string) {
	prefix := pterm.FgCyan.Sprintf("[%s]", lang)
	switch status {
	case "ok":
		pterm.Success.Println(prefix, details)
	case "skip":
		pterm.Warning.Println(prefix, details)
	case "error":
		pterm.Error.Println(prefix, details)
	case "info":
		pterm.Info.Println(prefix, details)
	default:
		fmt.Printf("%s %s\n", prefix, details)
	}
}

// Stats prints build statistics in a table.
func (u *UI) Stats(title string, stats map[string]interface{}) {
	pterm.DefaultSection.WithLevel(2).Println(title)

	var data [][]string
	for k, v := range stats {
		data = append(data, []string{k, fmt.Sprintf("%v", v)})
	}

	pterm.DefaultTable.WithData(data).Render()
	fmt.Println()
}

// LanguageStats prints per-language statistics.
func (u *UI) LanguageStats(byLanguage map[string]int) {
	if len(byLanguage) == 0 {
		return
	}

	data := pterm.TableData{{"Language", "Words"}}
	for lang, count := range byLanguage {
		data = append(data, []string{lang, fmt.Sprintf("%d", count)})
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	fmt.Println()
}

// LengthStats prints per-length statistics.
func (u *UI) LengthStats(byLength map[int]int) {
	if len(byLength) == 0 {
		return
	}

	data := pterm.TableData{{"Length", "Words"}}
	for length := 3; length <= 10; length++ {
		if count, ok := byLength[length]; ok && count > 0 {
			data = append(data, []string{
				fmt.Sprintf("%d-c", length),
				fmt.Sprintf("%d", count),
			})
		}
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
	fmt.Println()
}

// FinalReport prints the final summary report.
func (u *UI) FinalReport(totalWords int, filesWritten int, duration time.Duration) {
	pterm.DefaultSection.Println("Summary")

	panel := pterm.DefaultBox.WithTitle("Results").Sprint(
		fmt.Sprintf(
			"  Total Words:    %s\n"+
				"  Files Written:  %s\n"+
				"  Duration:       %s\n"+
				"  Throughput:     %s words/sec",
			pterm.FgGreen.Sprintf("%d", totalWords),
			pterm.FgCyan.Sprintf("%d", filesWritten),
			pterm.FgYellow.Sprint(duration.Round(time.Millisecond)),
			pterm.FgMagenta.Sprintf("%.0f", float64(totalWords)/duration.Seconds()),
		),
	)
	fmt.Println(panel)
}

// Success prints a success message.
func (u *UI) Success(message string) {
	pterm.Success.Println(message)
}

// Error prints an error message.
func (u *UI) Error(message string) {
	pterm.Error.Println(message)
}

// Warning prints a warning message.
func (u *UI) Warning(message string) {
	pterm.Warning.Println(message)
}

// Info prints an info message.
func (u *UI) Info(message string) {
	pterm.Info.Println(message)
}

// Debug prints a debug message (only in verbose mode).
func (u *UI) Debug(message string) {
	if u.verbose {
		pterm.Debug.Println(message)
	}
}

// Separator prints a visual separator.
func (u *UI) Separator() {
	pterm.DefaultBasicText.Println(pterm.FgGray.Sprint("─────────────────────────────────────────────────────────────"))
}

// Done prints the completion message.
func (u *UI) Done() {
	fmt.Println()
	pterm.DefaultCenter.Println(
		pterm.FgGreen.Sprint("✓ Done!"),
	)
}
