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

// Mode describes how the palette interprets the current input. The
// active mode is the first one in the configured list whose Match
// returns true; a nil Match matches anything, so it's typically used
// as the fallback (last entry).
type Mode struct {
	// Name identifies the mode for logging and host status displays.
	// Not rendered by the palette itself.
	Name string

	// Prompt is the glyph rendered before the input field. Should be
	// the same display width as the configured spinner so the input
	// text doesn't shift when the spinner swaps in during search. An
	// empty Prompt falls back to defaultPrompt ("◌ ").
	Prompt string

	// Match reports whether this mode applies to the given raw input.
	// A nil Match matches anything.
	Match func(input string) bool

	// Query extracts the meaningful query string from the raw input —
	// typically by stripping a leading prefix. A nil Query returns
	// the input unchanged.
	Query func(input string) string

	// Items returns the candidate items for this mode given the
	// palette state and the extracted query. A nil Items returns nil.
	Items func(m Model, query string) []Item
}

// defaultPrompt is the fallback prompt glyph for modes that don't set
// their own. Two cells wide to match the default spinner.
const defaultPrompt = "◌ "

// CommandMode is the default ">"-prefixed mode. Items are the
// configured commands, fuzzy-filtered by the query.
var CommandMode = Mode{
	Name:   "command",
	Prompt: defaultPrompt,
	Match: func(input string) bool {
		return strings.HasPrefix(input, ">")
	},
	Query: func(input string) string {
		return strings.TrimSpace(strings.TrimPrefix(input, ">"))
	},
	Items: func(m Model, q string) []Item {
		return FilterFuzzy(m.commands, q)
	},
}

// SearchMode is the default fallback. It matches any input not
// claimed by an earlier mode and surfaces the most recent async
// search results.
var SearchMode = Mode{
	Name:   "search",
	Prompt: defaultPrompt,
	Match:  nil, // nil = catch-all
	Query:  nil, // nil = identity
	Items:  func(m Model, _ string) []Item { return m.results },
}

// SearchFunc is the caller-provided async search. It returns a tea.Cmd
// that eventually yields a SearchResultMsg.
type SearchFunc func(query string) tea.Cmd

// Model is the palette bubble.
type Model struct {
	input     textinput.Model
	spinner   spinner.Model
	paginator paginator.Model
	help      help.Model

	modes    []Mode
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
	pg.ActiveDot = "● "
	pg.InactiveDot = "○ "

	m := Model{
		input:     ti,
		spinner:   sp,
		paginator: pg,
		help:      help.New(),
		modes:     []Mode{CommandMode, SearchMode},
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

	// Navigation and Execute keys are consumed by the palette and NOT
	// forwarded to the textinput, which would otherwise treat ↑/↓ as
	// suggestion navigation and Enter as input submission.
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kp, m.KeyMap.Down):
			m.moveCursor(1)
			return m, nil
		case key.Matches(kp, m.KeyMap.Up):
			m.moveCursor(-1)
			return m, nil
		case key.Matches(kp, m.KeyMap.NextPage):
			m.pageBy(1)
			return m, nil
		case key.Matches(kp, m.KeyMap.PrevPage):
			m.pageBy(-1)
			return m, nil
		case key.Matches(kp, m.KeyMap.Execute):
			return m, m.execute()
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

// execute builds the tea.Cmd dispatched when the user presses Enter
// on a highlighted item. Always emits a SelectedMsg so the host can
// react (close the palette, log, etc.); when the item is a Command
// with a non-nil Run, batches Run()'s cmd alongside. Returns nil when
// no item is selected.
func (m Model) execute() tea.Cmd {
	sel := m.Selected()
	if sel == nil {
		return nil
	}
	selectMsg := func() tea.Msg { return SelectedMsg{Item: sel} }
	if c, ok := sel.(Command); ok && c.Run != nil {
		if runCmd := c.Run(); runCmd != nil {
			return tea.Batch(selectMsg, runCmd)
		}
	}
	return selectMsg
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

// pageBy snaps the cursor to the start of the page delta away from
// the current one, wrapping at the first and last page. No-op when
// pagination is disabled (pageSize == 0) or there are no items.
func (m *Model) pageBy(delta int) {
	if m.pageSize <= 0 {
		return
	}
	n := len(m.Items())
	if n == 0 {
		return
	}
	totalPages := (n + m.pageSize - 1) / m.pageSize
	currentPage := m.cursor / m.pageSize
	targetPage := ((currentPage+delta)%totalPages + totalPages) % totalPages
	m.cursor = targetPage * m.pageSize
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

	// Pick the leading glyph: the spinner while a search is pending,
	// otherwise the active mode's prompt (or the global default).
	glyph := m.Mode().Prompt
	if glyph == "" {
		glyph = defaultPrompt
	}
	if m.loading {
		glyph = m.spinner.View()
	}

	// Size the textinput to the available row width so it doesn't
	// overflow the container.
	inner := m.InnerWidth()
	if inner > 0 {
		w := inner - lipgloss.Width(indent) - lipgloss.Width(glyph)
		if w < 1 {
			w = 1
		}
		m.input.SetWidth(w)
	}
	sections = append(sections, indent+glyph+m.input.View())

	items := m.Items()
	desiredRows := 0
	if m.pageSize > 0 {
		desiredRows = m.pageSize * m.delegate.Height()
	}

	// Slice items down to the current page when pagination is on.
	pageItems := items
	pageStart := 0
	totalPages := 1
	if m.pageSize > 0 && len(items) > 0 {
		totalPages = (len(items) + m.pageSize - 1) / m.pageSize
		currentPage := m.cursor / m.pageSize
		pageStart = currentPage * m.pageSize
		pageEnd := pageStart + m.pageSize
		if pageEnd > len(items) {
			pageEnd = len(items)
		}
		pageItems = items[pageStart:pageEnd]
	}

	if len(pageItems) > 0 || desiredRows > 0 {
		// Pass the inner width to the delegate so selected rows fill
		// the full row, and translate cursor to page-local space so
		// the delegate's "is this index selected?" check works against
		// the sliced view.
		rowModel := m
		rowModel.width = inner
		rowModel.cursor = m.cursor - pageStart

		var lines []string
		for i, item := range pageItems {
			var buf strings.Builder
			m.delegate.Render(&buf, rowModel, i, item)
			lines = append(lines, strings.Split(buf.String(), "\n")...)
		}

		// Pad or truncate to a stable height so the palette doesn't
		// jump between modes with different item counts.
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

	// Paginator footer — only when there's more than one page.
	if totalPages > 1 {
		m.paginator.TotalPages = totalPages
		m.paginator.Page = m.cursor / m.pageSize
		sections = append(sections, "", indent+m.paginator.View())
	}

	if m.showHelp {
		helpWidth := inner - lipgloss.Width(indent)
		if helpWidth > 0 {
			m.help.SetWidth(helpWidth)
		}
		helpLine := m.help.View(m)
		if helpLine != "" {
			sections = append(sections, "", indent+helpLine)
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

// Mode returns the currently active Mode — the first in the
// configured list whose Match returns true (or whose Match is nil).
// Returns a zero Mode when no modes are configured.
func (m Model) Mode() Mode {
	input := m.input.Value()
	for _, mode := range m.modes {
		if mode.Match == nil || mode.Match(input) {
			return mode
		}
	}
	return Mode{}
}

// Query returns the active mode's interpretation of the input value
// (typically with a leading prefix stripped). Falls back to the raw
// input when the active mode has no Query function.
func (m Model) Query() string {
	mode := m.Mode()
	if mode.Query == nil {
		return m.input.Value()
	}
	return mode.Query(m.input.Value())
}

// Value returns the raw input value.
func (m Model) Value() string { return m.input.Value() }

// Items returns the candidate items for the currently active mode.
// Returns nil when no mode is active or the mode declares no Items
// function.
func (m Model) Items() []Item {
	mode := m.Mode()
	if mode.Items == nil {
		return nil
	}
	return mode.Items(m, m.Query())
}

// FilterFuzzy returns items whose FilterValue is a fuzzy-subsequence
// match for query, ordered by relevance (best first). An empty query
// returns the input unchanged. Exported so custom Mode.Items closures
// can reuse the same filter logic as the built-in CommandMode.
func FilterFuzzy(items []Item, query string) []Item {
	if query == "" {
		return items
	}
	targets := make([]string, len(items))
	for i, c := range items {
		targets[i] = c.FilterValue()
	}
	matches := fuzzy.Find(query, targets)
	out := make([]Item, len(matches))
	for i, mt := range matches {
		out[i] = items[mt.Index]
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
