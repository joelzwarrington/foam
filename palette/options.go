package palette

import "charm.land/bubbles/v2/paginator"

// Option configures a palette Model. Apply with New(...Option).
type Option func(*Model)

// WithCommands seeds the command list shown in CommandMode.
func WithCommands(cmds []Item) Option {
	return func(m *Model) { m.commands = cmds }
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
