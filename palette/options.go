package palette

import "charm.land/bubbles/v2/paginator"

// Option configures a palette Model. Apply with New(...Option).
type Option func(*Model)

// WithCommands seeds the command list shown in CommandMode.
func WithCommands(cmds []Item) Option {
	return func(m *Model) { m.commands = cmds }
}

// WithModes replaces the default mode list (CommandMode, SearchMode)
// with the supplied modes, in priority order — the first whose Match
// returns true wins. Make the last entry a fallback (Match: nil) so
// some mode always applies.
func WithModes(modes ...Mode) Option {
	return func(m *Model) { m.modes = modes }
}

// WithTitle sets the optional section header rendered above the input.
// Pass an empty string (the default) for no title row.
func WithTitle(s string) Option {
	return func(m *Model) { m.title = s }
}

// WithHelp toggles the short-help row at the bottom of the palette.
// On by default.
func WithHelp(show bool) Option {
	return func(m *Model) { m.showHelp = show }
}

// WithSearch wires up the async search function used in SearchMode.
func WithSearch(fn SearchFunc) Option {
	return func(m *Model) { m.search = fn }
}

// WithDelegate overrides the ItemDelegate used to render items.
func WithDelegate(d ItemDelegate) Option {
	return func(m *Model) { m.delegate = d }
}

// WithKeyMap overrides the default keybindings.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) { m.KeyMap = km }
}

// WithStyles overrides the default visual styles.
func WithStyles(s Styles) Option {
	return func(m *Model) { m.Styles = s }
}

// WithPageSize sets a fixed number of items per page. Pass 0 to
// auto-fit to the available terminal height (the default).
func WithPageSize(n int) Option {
	return func(m *Model) {
		m.pageSize = n
		if n > 0 {
			m.paginator.PerPage = n
		}
	}
}

// WithPaginatorType selects between dot indicators and "1/N" numeric.
func WithPaginatorType(t paginator.Type) Option {
	return func(m *Model) { m.paginator.Type = t }
}
