// Command palette-simple is the minimal foam/palette example: a
// single mode that fuzzy-filters a list of commands as you type — no
// "> "-style prefix required. The first command demonstrates a
// Command.Run that returns tea.Quit; the others just emit
// SelectedMsg so you can see the dispatch round-trip.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/joelzwarrington/foam/palette"
)

// commands is the list this example fuzzy-filters. It lives at
// package scope so commandsMode's Items closure can close over it —
// the new style of seeding items directly via a Mode.
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

// commandsMode is the only mode in this example. Match is nil so it
// catches every input; the Items closure fuzzy-filters its own slice.
var commandsMode = palette.Mode{
	Name: "commands",
	Items: func(_ palette.Model, q string) []palette.Item {
		return palette.FilterFuzzy(commands, q)
	},
}

type model struct {
	palette palette.Model
	status  string
}

func initialModel() model {
	p := palette.New(
		palette.WithModes(commandsMode),
		palette.WithPlaceholder("Search for commands by name..."),
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
		}
	}

	var cmd tea.Cmd
	m.palette, cmd = m.palette.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	status := "\n  (ctrl+c to quit)"
	if m.status != "" {
		status = "\n  " + m.status + "    (ctrl+c to quit)"
	}
	v := tea.NewView(m.palette.View() + status)
	v.AltScreen = true
	return v
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
