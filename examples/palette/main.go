// Command palette-example runs the palette bubble standalone. Type to
// filter commands (prefix the input with ">"), use ↑/↓ to navigate,
// Enter to dispatch. Quit dispatches tea.Quit; the other commands set
// a status line so you can see the SelectedMsg round-trip. ctrl+c
// always exits.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/joelzwarrington/foam/palette"
)

type model struct {
	palette palette.Model
	status  string
}

func initialModel() model {
	p := palette.New(
		palette.WithCommands([]palette.Item{
			palette.Command{ID: "open", Name: "Open file", Desc: "Open a file in the editor"},
			palette.Command{ID: "save", Name: "Save", Desc: "Save the current buffer"},
			palette.Command{
				ID:   "quit",
				Name: "Quit",
				Desc: "Exit the program",
				Run:  func() tea.Cmd { return tea.Quit },
			},
		}),
		palette.WithPageSize(3),
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
	modeName := "search"
	if m.palette.Mode() == palette.CommandMode {
		modeName = "command"
	}
	status := fmt.Sprintf("\n\nmode: %s   query: %q", modeName, m.palette.Query())
	if m.status != "" {
		status += "   " + m.status
	}
	status += "   (ctrl+c to quit)"
	return tea.NewView(m.palette.View() + status)
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
