// Command palette-overlay shows the palette as a modal layer over a
// background model — the canonical "modal palette" recipe. Press
// ctrl+k to toggle the palette in and out; while hidden, key events
// land on the background; while visible, they go to the palette
// only.
//
// The background is a fake "editor" buffer drawn behind the palette
// using lipgloss.Canvas to composite the two layers.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/joelzwarrington/foam/palette"
)

// paletteWidth is the fixed outer width of the palette overlay,
// regardless of terminal size. The palette is centered horizontally.
const paletteWidth = 60

// fakeEditorBuffer is the static "background" content drawn behind
// the palette so the overlay has something to overlay on top of.
const fakeEditorBuffer = `package palette

// Mode describes how the palette interprets the current input. The
// active mode is the first one in the configured list whose Match
// returns true; a nil Match matches anything.
type Mode struct {
    Name     string
    Prompt   string
    Debounce time.Duration
    Match    func(input string) bool
    Query    func(input string) string
    Items    func(m Model, query string) []Item
    Search   func(ctx context.Context, query string) tea.Cmd
}

func (m Model) Mode() Mode {
    input := m.input.Value()
    for _, mode := range m.modes {
        if mode.Match == nil || mode.Match(input) {
            return mode
        }
    }
    return Mode{}
}

// — press ctrl+k to open the command palette —`

// commandsMode is the same single-mode setup as examples/palette/simple.
var commandsMode = palette.Mode{
	Name: "commands",
	Items: func(m palette.Model, q string) []palette.Item {
		return palette.FilterFuzzy(m.Commands(), q)
	},
}

type model struct {
	palette       palette.Model
	visible       bool
	width, height int
	status        string
}

func initialModel() model {
	p := palette.New(
		palette.WithModes(commandsMode),
		palette.WithCommands([]palette.Item{
			palette.Command{ID: "open", Name: "Open file", Desc: "Open a file in the editor"},
			palette.Command{ID: "save", Name: "Save", Desc: "Save the current buffer"},
			palette.Command{ID: "format", Name: "Format document", Desc: "Run the configured formatter"},
			palette.Command{ID: "theme", Name: "Change theme", Desc: "Toggle between light and dark"},
			palette.Command{
				ID:   "quit",
				Name: "Quit",
				Desc: "Exit the program",
				Run:  func() tea.Cmd { return tea.Quit },
			},
		}),
		palette.WithPageSize(5),
	)
	p.SetWidth(paletteWidth)
	return model{palette: p}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+k":
			m.visible = !m.visible
			if m.visible {
				return m, m.palette.Focus()
			}
			m.palette.Blur()
			m.palette.Reset()
			return m, nil
		}

	case palette.SelectedMsg:
		if c, ok := msg.Item.(palette.Command); ok {
			m.status = "ran: " + c.Name
		}
		// Auto-close on selection.
		m.visible = false
		m.palette.Blur()
		m.palette.Reset()
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Update(msg)
		return m, cmd
	}

	// Gate input: when the palette is visible, only the palette gets
	// keystrokes. Otherwise the background "owns" them — here it just
	// ignores them, but a real app would forward to its own model.
	if m.visible {
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		// Pre-WindowSizeMsg: just show the background unframed.
		v := tea.NewView(fakeEditorBuffer)
		v.AltScreen = true
		return v
	}

	bgStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2)

	hint := lipgloss.NewStyle().Faint(true).
		Render("ctrl+k: open palette   ctrl+c: quit")
	if m.status != "" {
		hint = lipgloss.NewStyle().Faint(true).Render(m.status) + "    " + hint
	}
	background := bgStyle.Render(fakeEditorBuffer + "\n\n" + hint)

	if !m.visible {
		v := tea.NewView(background)
		v.AltScreen = true
		return v
	}

	canvas := lipgloss.NewCanvas(m.width, m.height)
	canvas.Compose(lipgloss.NewLayer(background))
	canvas.Compose(
		lipgloss.NewLayer(m.palette.View()).
			X((m.width - paletteWidth) / 2).
			Y(m.height / 6),
	)
	v := tea.NewView(canvas.Render())
	v.AltScreen = true
	return v
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
