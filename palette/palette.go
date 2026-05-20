// Package palette provides a command-palette bubble for Bubble Tea
// programs. It has two modes derived from the input itself: input
// starting with ">" filters a predefined command list (CommandMode);
// anything else dispatches to a caller-provided async search
// (SearchMode).
package palette

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
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
	help      help.Model

	commands []Item
	results  []Item
	delegate ItemDelegate
	search   SearchFunc

	title    string
	cursor   int
	pageSize int
	loading  bool
	width    int
	height   int
	showHelp bool

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
		help:      help.New(),
		delegate:  NewDefaultDelegate(),
		showHelp:  true,
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

// Update handles cursor navigation, forwards remaining messages to
// the textinput, tracks terminal size, and resets the cursor whenever
// the input value changes (so it can't dangle past the end of a
// freshly filtered command list). Paging and Enter dispatch land in
// later milestones.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		m.height = ws.Height
	}

	// Navigation keys are consumed by the palette and NOT forwarded
	// to the textinput, which would otherwise treat them as
	// suggestion navigation.
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, m.KeyMap.Down):
			m.moveCursor(1)
			return m, nil
		case key.Matches(kp, m.KeyMap.Up):
			m.moveCursor(-1)
			return m, nil
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.cursor = 0
	}
	return m, cmd
}

// ShortHelp returns the compact key list rendered by the help bubble
// at the bottom of the palette. Combines Up/Down into a single
// synthetic "↑↓ navigate" entry for legibility; the actual KeyMap
// bindings remain split since they're separate actions.
func (m Model) ShortHelp() []key.Binding {
	// WithKeys is required even for display-only bindings — help
	// treats keyless bindings as disabled and skips them.
	nav := key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑↓", "navigate"),
	)
	return []key.Binding{nav, m.KeyMap.Execute, m.KeyMap.Cancel}
}

// FullHelp returns the expanded key groups for help bubbles displaying
// the full layout (not used by the palette itself by default, but
// available for hosts that wire up "?"-toggled help).
func (m Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.KeyMap.Up, m.KeyMap.Down},
		{m.KeyMap.PrevPage, m.KeyMap.NextPage},
		{m.KeyMap.Execute, m.KeyMap.Cancel},
	}
}

// moveCursor shifts the selection by delta, wrapping at both ends. A
// no-op when there are no items.
func (m *Model) moveCursor(delta int) {
	n := len(m.Items())
	if n == 0 {
		return
	}
	m.cursor = ((m.cursor+delta)%n + n) % n
}

// InnerWidth is the usable width inside the Container border/padding.
// Returns 0 when the outer width has not yet been set.
func (m Model) InnerWidth() int {
	if m.width <= 0 {
		return 0
	}
	inner := m.width - m.Styles.Container.GetHorizontalFrameSize()
	if inner < 0 {
		return 0
	}
	return inner
}

// View composes the palette layout: an optional title, the text
// input, and the visible items rendered through the configured
// ItemDelegate, wrapped in the Container style. Items are passed the
// inner width so the delegate's selection background can fill the row.
// The spinner row and paginator footer land in later milestones.
func (m Model) View() string {
	indent := m.Styles.Indent

	var sections []string
	if m.title != "" {
		sections = append(sections, indent+m.Styles.Title.Render(m.title), "")
	}

	// Size the textinput to the available row width so it doesn't
	// overflow the container.
	inner := m.InnerWidth()
	if inner > 0 {
		w := inner - lipgloss.Width(indent)
		if w < 1 {
			w = 1
		}
		m.input.SetWidth(w)
	}
	sections = append(sections, indent+m.input.View())

	items := m.Items()
	desiredRows := 0
	if m.pageSize > 0 {
		desiredRows = m.pageSize * m.delegate.Height()
	}
	if len(items) > 0 || desiredRows > 0 {
		// Pass the inner width to the delegate so selected rows fill
		// the full row. We mutate the local m (value receiver) — the
		// caller's copy is unaffected.
		rowModel := m
		rowModel.width = inner

		var lines []string
		for i, item := range items {
			var buf strings.Builder
			m.delegate.Render(&buf, rowModel, i, item)
			lines = append(lines, strings.Split(buf.String(), "\n")...)
		}

		// When pageSize is set, pad or truncate to a stable height so
		// the palette doesn't jump between modes with different item
		// counts. Pagination chooses which items make it in (later
		// milestone); here we just enforce the row budget.
		if desiredRows > 0 {
			for len(lines) < desiredRows {
				lines = append(lines, "")
			}
			if len(lines) > desiredRows {
				lines = lines[:desiredRows]
			}
		}

		sections = append(sections, "", strings.Join(lines, "\n"))
	}

	if m.showHelp {
		helpWidth := inner - lipgloss.Width(indent)
		if helpWidth > 0 {
			m.help.SetWidth(helpWidth)
		}
		helpLine := m.help.View(m)
		if helpLine != "" {
			sections = append(sections, "", indent+m.Styles.Footer.Render(helpLine))
		}
	}

	body := strings.Join(sections, "\n")

	// Pin the container's content width so lipgloss pads every row to
	// the same width — otherwise short rows (title, spacers) don't
	// reach the right border and it appears to be missing.
	container := m.Styles.Container
	if inner > 0 {
		container = container.Width(inner)
	}
	return container.Render(body)
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
// this is the predefined command list fuzzy-filtered by Query(); when
// the query is empty all commands are returned. In SearchMode it's
// the most recent search results, unfiltered locally.
func (m Model) Items() []Item {
	if m.Mode() == CommandMode {
		return filterCommands(m.commands, m.Query())
	}
	return m.results
}

// filterCommands runs a fuzzy match of query against each command's
// FilterValue and returns the matches ordered by relevance (best
// first). An empty query returns the input unchanged.
func filterCommands(cmds []Item, query string) []Item {
	if query == "" {
		return cmds
	}
	targets := make([]string, len(cmds))
	for i, c := range cmds {
		targets[i] = c.FilterValue()
	}
	matches := fuzzy.Find(query, targets)
	out := make([]Item, len(matches))
	for i, mt := range matches {
		out[i] = cmds[mt.Index]
	}
	return out
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
