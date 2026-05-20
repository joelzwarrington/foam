// Command palette-example runs the palette bubble standalone. It is a
// scratchpad for sanity-checking the palette skeleton: type into the
// input to see the derived Mode and Query update live; press ctrl+c
// to quit.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/joelzwarrington/foam/palette"
)

type model struct {
	palette palette.Model
}

func initialModel() model {
	p := palette.New(
		palette.WithCommands([]palette.Item{
			palette.Command{ID: "open", Name: "Open file", Desc: "Open a file in the editor"},
			palette.Command{ID: "save", Name: "Save", Desc: "Save the current buffer"},
			palette.Command{ID: "quit", Name: "Quit", Desc: "Exit the program"},
		}),
		palette.WithPageSize(3),
	)
	p.Focus()
	return model{palette: p}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "ctrl+c" {
		return m, tea.Quit
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
	status := fmt.Sprintf("\n\nmode: %s   query: %q   (ctrl+c to quit)", modeName, m.palette.Query())
	return tea.NewView(m.palette.View() + status)
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
