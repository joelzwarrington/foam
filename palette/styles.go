package palette

import "charm.land/lipgloss/v2"

// Styles holds the lipgloss styles the palette uses to render itself.
type Styles struct {
	// Container wraps the whole palette (input + spinner + items + footer).
	Container lipgloss.Style
	// Input wraps the text-input row.
	Input lipgloss.Style
	// SpinnerRow wraps the spinner + "Searching…" label shown while loading.
	SpinnerRow lipgloss.Style
	// SpinnerLabel styles the text next to the spinner glyph.
	SpinnerLabel lipgloss.Style
	// Footer wraps the paginator row at the bottom.
	Footer lipgloss.Style
}

// DefaultStyles returns sensible defaults. Override fields individually
// or pass a whole struct via WithStyles.
func DefaultStyles() Styles {
	return Styles{
		Container:    lipgloss.NewStyle(),
		Input:        lipgloss.NewStyle(),
		SpinnerRow:   lipgloss.NewStyle(),
		SpinnerLabel: lipgloss.NewStyle().Faint(true),
		Footer:       lipgloss.NewStyle().Faint(true),
	}
}
