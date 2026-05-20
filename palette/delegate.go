package palette

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Item is anything that can appear in the palette list. Both predefined
// commands and async search results implement it.
type Item interface {
	FilterValue() string
}

// DefaultItem is the convention DefaultDelegate knows how to render.
// Implement this if you want title/description rendering out of the box.
type DefaultItem interface {
	Item
	Title() string
	Description() string
}

// ItemDelegate controls how an Item is rendered and how key events
// reach the currently selected item. Mirrors bubbles/list.ItemDelegate.
type ItemDelegate interface {
	// Height is the number of terminal rows one item occupies.
	Height() int
	// Spacing is the number of blank rows between adjacent items.
	Spacing() int
	// Update receives messages while the delegate is active. Implement
	// item-level keybindings here, or return nil to opt out.
	Update(msg tea.Msg, m *Model) tea.Cmd
	// Render draws one item at the given visible index to w.
	Render(w io.Writer, m Model, index int, item Item)
}

// DelegateStyles holds the styles used by DefaultDelegate.
type DelegateStyles struct {
	Title            lipgloss.Style
	Description      lipgloss.Style
	SelectedTitle    lipgloss.Style
	SelectedDesc     lipgloss.Style
	SelectionMarker  string
	UnselectedMarker string
}

// DefaultDelegate renders DefaultItems with a title/description layout
// and selection highlighting. The actual rendering is implemented in a
// later milestone — this struct holds the configuration surface.
type DefaultDelegate struct {
	Styles DelegateStyles
}

// NewDefaultDelegate returns a DefaultDelegate with sensible defaults.
func NewDefaultDelegate() DefaultDelegate {
	return DefaultDelegate{
		Styles: DelegateStyles{
			Title:            lipgloss.NewStyle(),
			Description:      lipgloss.NewStyle().Faint(true),
			SelectedTitle:    lipgloss.NewStyle().Bold(true),
			SelectedDesc:     lipgloss.NewStyle(),
			SelectionMarker:  "▸ ",
			UnselectedMarker: "  ",
		},
	}
}

// Height reports one row per item by default.
func (d DefaultDelegate) Height() int { return 1 }

// Spacing reports zero blank rows between items by default.
func (d DefaultDelegate) Spacing() int { return 0 }

// Update is a no-op by default. Override by wrapping or replacing.
func (d DefaultDelegate) Update(_ tea.Msg, _ *Model) tea.Cmd { return nil }

// Render is a placeholder. The real layout (selection marker, title,
// description, truncation) lands in the delegate milestone.
func (d DefaultDelegate) Render(w io.Writer, m Model, index int, item Item) {
	marker := d.Styles.UnselectedMarker
	if index == m.cursor {
		marker = d.Styles.SelectionMarker
	}
	if di, ok := item.(DefaultItem); ok {
		_, _ = fmt.Fprintf(w, "%s%s\n", marker, di.Title())
		return
	}
	_, _ = fmt.Fprintf(w, "%s%s\n", marker, item.FilterValue())
}
