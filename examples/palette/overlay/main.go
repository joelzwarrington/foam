// Command palette-overlay shows the palette as a modal layer over a
// background model — the canonical "modal palette" recipe. Press
// ctrl+k to toggle the palette in and out; while hidden, key events
// land on the background; while visible, they go to the palette
// only.
//
// The background is a Markdown document rendered with Glamour and
// composited under the palette via lipgloss.Canvas.
package main

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/joelzwarrington/foam/palette"
)

// paletteWidth is the fixed outer width of the palette overlay,
// regardless of terminal size. The palette is centered horizontally.
const paletteWidth = 60

// backgroundMarkdown is the static document Glamour renders behind
// the palette. Picked to exercise headings, lists, blockquotes, code
// fences, and links — the things Glamour shows off.
const backgroundMarkdown = `# Lorem ipsum dolor sit amet

Consectetur **adipiscing elit**, sed do _eiusmod tempor_ incididunt ut
labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud
exercitation [ullamco laboris](https://example.com) nisi ut aliquip ex
ea commodo consequat.

## Duis aute irure dolor

- in reprehenderit in voluptate
- velit esse cillum dolore eu fugiat
- nulla pariatur excepteur sint occaecat

> Sed ut perspiciatis unde omnis iste natus error sit voluptatem
> accusantium doloremque laudantium, totam rem aperiam.

` + "```go\nfunc main() {\n    fmt.Println(\"hello, foam\")\n}\n```" + `
`

// hostKeys are the background-level bindings shown by the help bubble
// at the bottom of the screen when the palette is dismissed.
type hostKeys struct {
	OpenPalette key.Binding
	Quit        key.Binding
}

func (k hostKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.OpenPalette, k.Quit}
}

func (k hostKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.OpenPalette, k.Quit}}
}

var defaultHostKeys = hostKeys{
	OpenPalette: key.NewBinding(key.WithKeys("ctrl+k"), key.WithHelp("ctrl+k", "palette")),
	Quit:        key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
}

// commands is the list this example fuzzy-filters.
var commands = []palette.Item{
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
}

// commandsMode is the same single-mode setup as examples/palette/simple —
// catch-all Match, fuzzy-filter against the package-level commands slice.
var commandsMode = palette.Mode{
	Name: "commands",
	Items: func(_ palette.Model, q string) []palette.Item {
		return palette.FilterFuzzy(commands, q)
	},
}

type model struct {
	palette       palette.Model
	help          help.Model
	background    string // Glamour-rendered markdown, computed once at startup
	visible       bool
	width, height int
	status        string
}

func initialModel() model {
	p := palette.New(
		palette.WithModes(commandsMode),
		palette.WithPageSize(5),
	)
	p.SetWidth(paletteWidth)

	rendered, err := glamour.Render(backgroundMarkdown, "dark")
	if err != nil {
		// Fall back to raw markdown so the demo still launches.
		rendered = backgroundMarkdown
	}
	return model{palette: p, help: help.New(), background: rendered}
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
		case "esc":
			// Esc dismisses the palette overlay. The palette's own help
			// row advertises "esc cancel"; the host wires the actual
			// close.
			if m.visible {
				m.visible = false
				m.palette.Blur()
				m.palette.Reset()
				return m, nil
			}
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
		// Pre-WindowSizeMsg: just show the rendered markdown unframed.
		v := tea.NewView(m.background)
		v.AltScreen = true
		return v
	}

	m.help.SetWidth(m.width)
	helpLine := m.help.View(defaultHostKeys)
	if m.status != "" {
		helpLine = lipgloss.NewStyle().Faint(true).Render(m.status) + "    " + helpLine
	}

	background := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(m.background + "\n" + helpLine)

	if !m.visible {
		v := tea.NewView(background)
		v.AltScreen = true
		return v
	}

	// Pin the palette near the top of the screen so the markdown
	// beneath it stays visible. lipgloss.NewCompositor is required for
	// layer X/Y positioning — Canvas.Compose(layer) directly would
	// clear the whole canvas on each call and ignore the layer's
	// offset.
	paletteLayer := lipgloss.NewLayer(m.palette.View()).
		X((m.width - paletteWidth) / 2).
		Y(1)
	canvas := lipgloss.NewCanvas(m.width, m.height)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(background),
		paletteLayer,
	))
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
