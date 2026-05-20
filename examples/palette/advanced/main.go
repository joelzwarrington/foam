// Command palette-example runs the palette bubble standalone. Prefixes:
//
//	>  command palette (sync, filters local commands)
//	@  file picker    (sync, instant filter as you type)
//	(nothing)  web search (async, 100ms debounce + ~400ms round-trip)
//
// Web search also supports the kind: facet — type "kind:" inside the
// query to filter results by type (doc / repo / site). Multiple facets
// stack with free text: "tui kind:repo".
//
// Use ↑/↓ to navigate, Enter to dispatch, ctrl+c to quit.
package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/joelzwarrington/foam/palette"
)

// fileItem is a search-result item the file picker emits.
type fileItem struct{ name string }

func (f fileItem) FilterValue() string { return f.name }
func (f fileItem) Title() string       { return f.name }
func (f fileItem) Description() string { return "file" }

// webPage is a fake search-engine result.
type webPage struct{ title, url, kind string }

func (w webPage) FilterValue() string { return w.title }
func (w webPage) Title() string       { return w.title }
func (w webPage) Description() string { return w.url }

// kindValue renders one entry in the kind: facet completer.
type kindValue struct {
	name, desc string
}

func (k kindValue) FilterValue() string { return k.name }
func (k kindValue) Title() string       { return k.name }
func (k kindValue) Description() string { return k.desc }

// files is the corpus the "@" file picker searches against
var files = []string{
	".github/workflows/ci.yaml",
	".golangci.yml",
	"Brewfile",
	"LICENSE",
	"README.md",
	"Taskfile.yml",
	"examples/palette/main.go",
	"foam.go",
	"go.mod",
	"go.sum",
	"palette/command.go",
	"palette/delegate.go",
	"palette/delegate_test.go",
	"palette/keymap.go",
	"palette/messages.go",
	"palette/options.go",
	"palette/palette.go",
	"palette/palette_test.go",
	"palette/styles.go",
}

// pages is the corpus the search mode looks through.
var pages = []webPage{
	{"Bubble Tea — A powerful TUI framework", "github.com/charmbracelet/bubbletea", "repo"},
	{"Bubbles — Pre-built TUI components", "github.com/charmbracelet/bubbles", "repo"},
	{"Lip Gloss — Style definitions for TUIs", "github.com/charmbracelet/lipgloss", "repo"},
	{"The Elm Architecture", "guide.elm-lang.org/architecture/", "doc"},
	{"Go: A modern programming language", "go.dev", "site"},
	{"Charm — Beautiful tools for the terminal", "charm.sh", "site"},
	{"VS Code Command Palette docs", "code.visualstudio.com/docs/getstarted/userinterface", "doc"},
	{"Sublime Text \"Goto Anything\"", "www.sublimetext.com/docs/goto_anything.html", "doc"},
}

// kindFacet completes the "kind:" filter in web search. Sync — values
// are a small fixed set.
var kindFacet = palette.Facet{
	Name: "kind",
	Desc: "kind: filter results by type",
	Items: func(partial string) []palette.Item {
		kinds := []kindValue{
			{"doc", "documentation"},
			{"repo", "source code"},
			{"site", "homepage / landing page"},
		}
		out := make([]palette.Item, 0, len(kinds))
		needle := strings.ToLower(partial)
		for _, k := range kinds {
			if partial == "" || strings.HasPrefix(k.name, needle) {
				out = append(out, k)
			}
		}
		return out
	},
}

// filesMode is purely synchronous: each keystroke re-filters the
// hardcoded corpus inline, so results update instantly with no
// debounce and no spinner.
var filesMode = palette.Mode{
	Name:  "files",
	Match: func(s string) bool { return strings.HasPrefix(s, "@") },
	Query: func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, "@")) },
	Items: func(_ palette.Model, q string) []palette.Item {
		items := make([]palette.Item, 0)
		needle := strings.ToLower(q)
		for _, name := range files {
			if q == "" || strings.Contains(strings.ToLower(name), needle) {
				items = append(items, fileItem{name: name})
			}
		}
		return items
	},
}

// commands is the list the ">"-prefixed commandsMode fuzzy-filters.
var commands = []palette.Item{
	palette.Command{ID: "open", Name: "Open file", Desc: "Open a file in the editor"},
	palette.Command{ID: "new", Name: "New file", Desc: "Create an empty buffer"},
	palette.Command{ID: "save", Name: "Save", Desc: "Save the current buffer"},
	palette.Command{ID: "save-as", Name: "Save As…", Desc: "Save the buffer to a new path"},
	palette.Command{ID: "close", Name: "Close file", Desc: "Close the current buffer"},
	palette.Command{ID: "find", Name: "Find in files", Desc: "Search across the project"},
	palette.Command{ID: "format", Name: "Format document", Desc: "Run the configured formatter"},
	palette.Command{ID: "terminal", Name: "Toggle terminal", Desc: "Show or hide the integrated terminal"},
	palette.Command{ID: "sidebar", Name: "Toggle sidebar", Desc: "Show or hide the file tree"},
	palette.Command{ID: "reload", Name: "Reload window", Desc: "Restart the editor window"},
	palette.Command{
		ID:   "quit",
		Name: "Quit",
		Desc: "Exit the program",
		Run:  func() tea.Cmd { return tea.Quit },
	},
}

// commandsMode is the ">"-prefixed sync command picker.
var commandsMode = palette.Mode{
	Name:  "command",
	Match: func(s string) bool { return strings.HasPrefix(s, ">") },
	Query: func(s string) string { return strings.TrimSpace(strings.TrimPrefix(s, ">")) },
	Items: func(_ palette.Model, q string) []palette.Item {
		return palette.FilterFuzzy(commands, q)
	},
}

// searchMode is the catch-all (no prefix) and does a debounced async
// web search with a 400 ms simulated round-trip, demonstrating the
// spinner and the kind: facet.
var searchMode = palette.Mode{
	Name:     "search",
	Debounce: 100 * time.Millisecond,
	Match:    nil, // catch-all
	Query:    nil, // identity
	Facets:   []palette.Facet{kindFacet},
	Items: func(m palette.Model, _ string) []palette.Item {
		return m.Results("search")
	},
	Search: func(ctx context.Context, q string) tea.Cmd {
		return tea.Tick(400*time.Millisecond, func(_ time.Time) tea.Msg {
			if ctx.Err() != nil {
				return nil
			}
			text, parsed := palette.ParseFacets(q, []palette.Facet{kindFacet})
			kinds := parsed["kind"]
			items := make([]palette.Item, 0)
			needle := strings.ToLower(text)
			for _, p := range pages {
				if len(kinds) > 0 && !slices.Contains(kinds, p.kind) {
					continue
				}
				if text == "" ||
					strings.Contains(strings.ToLower(p.title), needle) ||
					strings.Contains(strings.ToLower(p.url), needle) {
					items = append(items, p)
				}
			}
			return palette.SearchResultMsg{Mode: "search", Query: q, Results: items}
		})
	},
}

type model struct {
	palette palette.Model
	status  string
}

func initialModel() model {
	p := palette.New(
		palette.WithModes(commandsMode, filesMode, searchMode),
		palette.WithPageSize(4),
	)
	p.Focus()
	return model{palette: p}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case palette.SelectedMsg:
		if c, ok := msg.Item.(palette.Command); ok {
			m.status = "ran: " + c.Name
		} else {
			m.status = "picked: " + msg.Item.FilterValue()
		}
	}

	var cmd tea.Cmd
	m.palette, cmd = m.palette.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	loading := ""
	if m.palette.Loading() {
		loading = "   loading…"
	}
	status := fmt.Sprintf("\n\nmode: %s   query: %q%s", m.palette.Mode().Name, m.palette.Query(), loading)
	if m.status != "" {
		status += "   " + m.status
	}
	status += "   (ctrl+c to quit)"
	v := tea.NewView(m.palette.View() + status)
	v.AltScreen = true
	// Cell-motion mouse capture is enough for the wheel events the
	// palette listens for.
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
