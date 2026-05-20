package palette

// SearchResultMsg is the message a SearchFunc eventually emits with
// the items matching a given query. Err is non-nil if the search
// failed; Results should be ignored in that case.
type SearchResultMsg struct {
	Query   string
	Results []Item
	Err     error
}

// SelectedMsg is dispatched when the user presses Execute (Enter by
// default) on a highlighted item. The host program type-switches on
// Item to decide how to react — close the palette, log, navigate,
// etc. When the item is a Command with a non-nil Run, the palette
// also fires Run()'s tea.Cmd in the same batch.
type SelectedMsg struct {
	Item Item
}
