package palette

import "charm.land/lipgloss/v2"

// Styles holds the lipgloss styles the palette uses to render itself.
type Styles struct {
	// Container wraps the whole palette. The default is a rounded
	// border with no padding — padding is applied manually as a
	// per-line indent so selection backgrounds can fill the full row.
	Container lipgloss.Style
	// Title styles the optional section header at the top of the
	// palette (see WithTitle).
	Title lipgloss.Style
	// Indent is the per-line left margin inside the container, as a
	// literal string. Two spaces by default.
	Indent string
	// SpinnerLabel styles the text next to the spinner glyph while a
	// search is in flight.
	SpinnerLabel lipgloss.Style
	// FacetHeader styles the facet-completion hint shown in the footer
	// row while the palette is completing a facet token.
	FacetHeader lipgloss.Style
	// Footer wraps the paginator row at the bottom.
	Footer lipgloss.Style
}

// DefaultStyles returns sensible defaults. Override fields individually
// or pass a whole struct via WithStyles.
func DefaultStyles() Styles {
	return Styles{
		Container:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
		Title:        lipgloss.NewStyle().Bold(true),
		Indent:       "  ",
		SpinnerLabel: lipgloss.NewStyle().Faint(true),
		FacetHeader:  lipgloss.NewStyle().Faint(true),
		Footer:       lipgloss.NewStyle().Faint(true),
	}
}
