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

// commandsMode is the only mode in this example. Match is nil so it
// catches every input; the Items closure runs the same fuzzy filter
// the built-in CommandMode would, but over the host-configured
// command list (m.Commands()).
var commandsMode = palette.Mode{
	Name: "commands",
	Items: func(m palette.Model, q string) []palette.Item {
		return palette.FilterFuzzy(m.Commands(), q)
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
		palette.WithCommands([]palette.Item{
			palette.Command{
				ID:   "quit",
				Name: "Quit",
				Desc: "Exit the program",
				Run:  func() tea.Cmd { return tea.Quit },
			},
			palette.Command{ID: "open", Name: "Open file", Desc: "Open a file in the editor"},
			palette.Command{ID: "save", Name: "Save", Desc: "Save the current buffer"},
			palette.Command{ID: "find", Name: "Find in files", Desc: "Search across the project"},
			palette.Command{ID: "format", Name: "Format document", Desc: "Run the configured formatter"},
		}),
		palette.WithPageSize(5),
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
