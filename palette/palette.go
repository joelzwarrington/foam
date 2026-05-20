// Package palette provides a command-palette bubble for Bubble Tea
// programs. It has two modes derived from the input itself: input
// starting with ">" filters a predefined command list (CommandMode);
// anything else dispatches to a caller-provided async search
// (SearchMode).
package palette

import (
	"context"
	"strings"
	"time"

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
	// Also used as the cache key for Search results — see
	// Model.Results.
	Name string

	// Prompt is the glyph rendered before the input field. Should be
	// the same display width as the configured spinner so the input
	// text doesn't shift when the spinner swaps in during search. An
	// empty Prompt falls back to defaultPrompt ("◌ ").
	Prompt string

	// Debounce is how long the palette waits after the input stops
	// changing before invoking Search. Zero means dispatch on the
	// next tick. Ignored when Search is nil.
	Debounce time.Duration

	// Match reports whether this mode applies to the given raw input.
	// A nil Match matches anything.
	Match func(input string) bool

	// Query extracts the meaningful query string from the raw input —
	// typically by stripping a leading prefix. A nil Query returns
	// the input unchanged.
	Query func(input string) string

	// Items returns the candidate items for this mode given the
	// palette state and the extracted query. For sync modes it does
	// the filtering inline; for async modes it typically reads from
	// the palette's Results cache that Search populates. A nil Items
	// returns nil.
	Items func(m Model, query string) []Item

	// Search is the async dispatcher. When the input changes inside
	// this mode and the debounce window elapses, the palette calls
	// Search and the returned tea.Cmd must eventually yield a
	// SearchResultMsg with the matching Mode name. The ctx is
	// cancelled when a newer search supersedes this one or when the
	// active mode changes — implementations should pass it through
	// to their HTTP/DB call. Nil means the mode is purely synchronous.
	Search func(ctx context.Context, query string) tea.Cmd
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
// claimed by an earlier mode and reads from the palette's cached
// results bucket for its Name. Hosts override Search via WithModes
// to actually populate results.
var SearchMode = Mode{
	Name:     "search",
	Prompt:   defaultPrompt,
	Debounce: 150 * time.Millisecond,
	Match:    nil, // nil = catch-all
	Query:    nil, // nil = identity
	Items: func(m Model, _ string) []Item {
		return m.results["search"]
	},
}

// debounceMsg is an internal tick that fires after a mode's Debounce
// window. The palette dispatches the mode's Search closure only when
// the generation hasn't moved on (no newer keystroke).
type debounceMsg struct {
	mode string
	gen  int
}

// Model is the palette bubble.
type Model struct {
	input     textinput.Model
	spinner   spinner.Model
	paginator paginator.Model
	help      help.Model

	modes    []Mode
	commands []Item
	results  map[string][]Item // per-mode cache: keyed by Mode.Name
	delegate ItemDelegate

	// search machinery
	searchGen    int                // increments on each input change for stale-tick rejection
	searchCancel context.CancelFunc // cancels the in-flight Search context

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
		results:   map[string][]Item{},
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

// Update handles cursor navigation, the async search lifecycle
// (debounce → dispatch → result), spinner ticks, and forwards
// remaining messages to the textinput.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case debounceMsg:
		return m.handleDebounce(msg)

	case SearchResultMsg:
		return m.handleSearchResult(msg)

	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		// Navigation and Execute keys are consumed by the palette and
		// NOT forwarded to the textinput.
		switch {
		case key.Matches(msg, m.KeyMap.Down):
			m.moveCursor(1)
			return m, nil
		case key.Matches(msg, m.KeyMap.Up):
			m.moveCursor(-1)
			return m, nil
		case key.Matches(msg, m.KeyMap.NextPage):
			m.pageBy(1)
			return m, nil
		case key.Matches(msg, m.KeyMap.PrevPage):
			m.pageBy(-1)
			return m, nil
		case key.Matches(msg, m.KeyMap.Execute):
			return m, m.execute()
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.cursor = 0
		if searchCmd := m.scheduleSearch(); searchCmd != nil {
			return m, tea.Batch(cmd, searchCmd)
		}
	}
	return m, cmd
}

// scheduleSearch is called whenever the input value changes. It
// cancels any in-flight Search and, if the now-active mode has a
// Search closure, schedules a debounce tick that will dispatch it.
func (m *Model) scheduleSearch() tea.Cmd {
	// Cancel any in-flight search — we're either coalescing keystrokes
	// or switching away from the mode that started it.
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	m.loading = false

	mode := m.Mode()
	if mode.Search == nil {
		return nil
	}
	m.searchGen++
	gen := m.searchGen
	name := mode.Name
	d := mode.Debounce
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return debounceMsg{mode: name, gen: gen}
	})
}

// handleDebounce dispatches the active mode's Search closure when the
// debounce tick is still current (no newer keystroke superseded it
// and the user hasn't switched mode).
func (m Model) handleDebounce(msg debounceMsg) (Model, tea.Cmd) {
	if msg.gen != m.searchGen {
		return m, nil // stale: a newer keystroke is pending
	}
	mode := m.Mode()
	if mode.Name != msg.mode || mode.Search == nil {
		return m, nil // mode switched out from under this tick
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.searchCancel = cancel
	m.loading = true
	return m, tea.Batch(mode.Search(ctx, m.Query()), m.spinner.Tick)
}

// handleSearchResult stores result items in the per-mode cache and
// clears loading. Stale results (whose Mode or Query no longer
// matches the current state) are dropped.
func (m Model) handleSearchResult(msg SearchResultMsg) (Model, tea.Cmd) {
	mode := m.Mode()
	if msg.Mode != mode.Name || msg.Query != m.Query() {
		return m, nil // stale
	}
	if m.results == nil {
		m.results = map[string][]Item{}
	}
	m.results[msg.Mode] = msg.Results
	m.loading = false
	return m, nil
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

// Results returns the cached items for the named mode (typically
// populated by that mode's Search closure via SearchResultMsg). A
// mode's Items closure usually reads from here.
func (m Model) Results(modeName string) []Item {
	return m.results[modeName]
}

// Commands returns the command list configured via WithCommands.
// Useful when a custom Mode wants to filter the same list using
// different prefix or matching semantics.
func (m Model) Commands() []Item { return m.commands }

// Loading reports whether a Search is currently in flight.
func (m Model) Loading() bool { return m.loading }

// SetWidth overrides the palette's outer width. Useful when the
// palette is laid out manually (e.g., as a fixed-width modal overlay)
// rather than filling the host's WindowSizeMsg.
func (m *Model) SetWidth(w int) { m.width = w }

// SetHeight overrides the palette's outer height. Currently advisory —
// pagination is driven by WithPageSize.
func (m *Model) SetHeight(h int) { m.height = h }

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

// Reset clears the input and result state, and cancels any in-flight
// Search.
func (m *Model) Reset() {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	m.input.SetValue("")
	m.results = map[string][]Item{}
	m.cursor = 0
	m.paginator.Page = 0
	m.loading = false
}
