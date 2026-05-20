// Package palette provides a command-palette bubble for Bubble Tea
// programs. It has two modes derived from the input itself: input
// starting with ">" filters a predefined command list (CommandMode);
// anything else dispatches to a caller-provided async search
// (SearchMode).
package palette

import (
	"strings"

	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// Mode is derived from the input value.
type Mode int

const (
	// SearchMode is the default — input does not start with ">".
	SearchMode Mode = iota
	// CommandMode is active when the input starts with ">".
	CommandMode
)

// SearchFunc is the caller-provided async search. It returns a tea.Cmd
// that eventually yields a SearchResultMsg.
type SearchFunc func(query string) tea.Cmd

// Model is the palette bubble.
type Model struct {
	input     textinput.Model
	spinner   spinner.Model
	paginator paginator.Model

	commands []Item
	results  []Item
	delegate ItemDelegate
	search   SearchFunc

	cursor   int
	pageSize int
	loading  bool
	width    int
	height   int

	KeyMap KeyMap
	Styles Styles
}

// New constructs a palette Model with sensible defaults. Apply Options
// to override.
func New(opts ...Option) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Type to search, or > for commands"

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	pg := paginator.New()
	pg.Type = paginator.Dots

	m := Model{
		input:     ti,
		spinner:   sp,
		paginator: pg,
		delegate:  NewDefaultDelegate(),
		KeyMap:    DefaultKeyMap(),
		Styles:    DefaultStyles(),
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

// Init is part of the tea.Model contract. The palette emits no startup
// command — callers compose it into their own program's Init.
func (m Model) Init() tea.Cmd { return nil }

// Update is a placeholder pass-through to the textinput, plus
// terminal-size tracking. Message routing (mode switching, debouncing,
// spinner ticks, pagination keys) is implemented in later milestones.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the text input followed by the visible items, each
// drawn through the configured ItemDelegate. The spinner row and
// paginator footer land in later milestones; until then this is the
// full output.
func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.input.View())

	items := m.Items()
	if len(items) == 0 {
		return b.String()
	}
	b.WriteString("\n")
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		m.delegate.Render(&b, m, i, item)
	}
	return b.String()
}

// Focus directs keyboard input to the palette.
func (m *Model) Focus() tea.Cmd { return m.input.Focus() }

// Blur removes keyboard focus from the palette.
func (m *Model) Blur() { m.input.Blur() }

// Mode is derived from the input value.
func (m Model) Mode() Mode {
	if strings.HasPrefix(m.input.Value(), ">") {
		return CommandMode
	}
	return SearchMode
}

// Query returns the input with the leading ">" stripped in CommandMode.
func (m Model) Query() string {
	v := m.input.Value()
	if strings.HasPrefix(v, ">") {
		return strings.TrimSpace(v[1:])
	}
	return v
}

// Value returns the raw input value.
func (m Model) Value() string { return m.input.Value() }

// Items returns the currently visible (filtered) items. In CommandMode
// this is the predefined command list; in SearchMode it's the most
// recent search results.
func (m Model) Items() []Item {
	if m.Mode() == CommandMode {
		return m.commands
	}
	return m.results
}

// Selected returns the highlighted item, or nil if none.
func (m Model) Selected() Item {
	items := m.Items()
	if m.cursor < 0 || m.cursor >= len(items) {
		return nil
	}
	return items[m.cursor]
}

// Page returns the current page (0-indexed).
func (m Model) Page() int { return m.paginator.Page }

// TotalPages returns the number of pages.
func (m Model) TotalPages() int { return m.paginator.TotalPages }

// Reset clears the input and result state.
func (m *Model) Reset() {
	m.input.SetValue("")
	m.results = nil
	m.cursor = 0
	m.paginator.Page = 0
	m.loading = false
}
