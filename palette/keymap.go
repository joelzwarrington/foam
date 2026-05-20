package palette

import "charm.land/bubbles/v2/key"

// KeyMap holds the palette's keybindings. Override individual fields
// to remap; pass the whole struct via WithKeyMap to swap wholesale.
type KeyMap struct {
	Execute  key.Binding
	Cancel   key.Binding
	Down     key.Binding
	Up       key.Binding
	NextPage key.Binding
	PrevPage key.Binding
}

// DefaultKeyMap returns the standard keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Execute: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "up"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "ctrl+f", "pgdown"),
			key.WithHelp("→", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "ctrl+b", "pgup"),
			key.WithHelp("←", "prev page"),
		),
	}
}
