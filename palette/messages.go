package palette

// SearchResultMsg is the message a SearchFunc eventually emits with
// the items matching a given query. Err is non-nil if the search
// failed; Results should be ignored in that case.
type SearchResultMsg struct {
	Query   string
	Results []Item
	Err     error
}
