package palette

import tea "charm.land/bubbletea/v2"

// Command is a built-in Item for use in CommandMode. Implements
// DefaultItem so the DefaultDelegate renders it without extra work.
type Command struct {
	ID   string
	Name string
	Desc string
	// Run is invoked when the user selects this command and presses
	// Enter. It may return nil if there's nothing to dispatch.
	Run func() tea.Cmd
}

// FilterValue is what fuzzy matching is performed against.
func (c Command) FilterValue() string { return c.Name }

// Title is rendered as the primary line by DefaultDelegate.
func (c Command) Title() string { return c.Name }

// Description is rendered as the secondary line by DefaultDelegate.
func (c Command) Description() string { return c.Desc }
