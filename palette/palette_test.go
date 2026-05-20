package palette

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
)

// testItem is a minimal Item used to populate the model in tests.
type testItem struct{ name string }

func (t testItem) FilterValue() string { return t.name }

func TestModeAndQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode Mode
		wantQry  string
	}{
		{"empty", "", SearchMode, ""},
		{"plain text", "foo", SearchMode, "foo"},
		{"just bracket", ">", CommandMode, ""},
		{"bracket space", "> ", CommandMode, ""},
		{"command", "> open", CommandMode, "open"},
		{"command no space", ">open", CommandMode, "open"},
		{"bracket inside", "foo > bar", SearchMode, "foo > bar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			m.input.SetValue(tc.input)

			if got := m.Mode(); got != tc.wantMode {
				t.Errorf("Mode() = %v, want %v", got, tc.wantMode)
			}
			if got := m.Query(); got != tc.wantQry {
				t.Errorf("Query() = %q, want %q", got, tc.wantQry)
			}
			if got := m.Value(); got != tc.input {
				t.Errorf("Value() = %q, want %q", got, tc.input)
			}
		})
	}
}

func TestItemsReflectsMode(t *testing.T) {
	cmds := []Item{Command{Name: "open"}, Command{Name: "close"}}
	results := []Item{testItem{name: "hit"}}

	m := New(WithCommands(cmds))
	m.results = results

	m.input.SetValue("foo")
	if got := m.Items(); len(got) != 1 || got[0].FilterValue() != "hit" {
		t.Errorf("SearchMode Items() = %v, want results", got)
	}

	// Empty query in CommandMode returns the full command list, in
	// declared order. Filtering is covered separately.
	m.input.SetValue(">")
	if got := m.Items(); len(got) != 2 || got[0].FilterValue() != "open" {
		t.Errorf("CommandMode Items() = %v, want commands", got)
	}
}

func TestCommandModeFuzzyFilter(t *testing.T) {
	cmds := []Item{
		Command{Name: "Open file"},
		Command{Name: "Save"},
		Command{Name: "Open in new tab"},
		Command{Name: "Quit"},
	}

	tests := []struct {
		name  string
		query string
		want  []string // expected FilterValues in order
	}{
		{
			name:  "empty query returns all in declared order",
			query: "",
			want:  []string{"Open file", "Save", "Open in new tab", "Quit"},
		},
		{
			name:  "exact substring matches",
			query: "open",
			want:  []string{"Open file", "Open in new tab"},
		},
		{
			name:  "case insensitive",
			query: "QuI",
			want:  []string{"Quit"},
		},
		{
			name:  "fuzzy non-contiguous chars",
			query: "qt", // Q...uiT
			want:  []string{"Quit"},
		},
		{
			name:  "no matches returns empty",
			query: "zzzz",
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New(WithCommands(cmds))
			m.input.SetValue(">" + tc.query)

			got := m.Items()
			if len(got) != len(tc.want) {
				t.Fatalf("Items() len = %d, want %d\ngot:  %v\nwant: %v", len(got), len(tc.want), itemNames(got), tc.want)
			}
			for i, w := range tc.want {
				if got[i].FilterValue() != w {
					t.Errorf("Items()[%d] = %q, want %q (full got: %v)", i, got[i].FilterValue(), w, itemNames(got))
				}
			}
		})
	}
}

func TestCursorResetsOnInputChange(t *testing.T) {
	m := New(WithCommands([]Item{
		Command{Name: "Open file"},
		Command{Name: "Save"},
		Command{Name: "Quit"},
	}))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 2

	// Simulate a keypress that mutates the input value.
	m, _ = m.Update(typeKey(t, "a"))

	if m.cursor != 0 {
		t.Errorf("cursor = %d after input change, want 0", m.cursor)
	}
}

func TestCursorUnchangedWhenInputUnchanged(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "a"}, Command{Name: "b"}, Command{Name: "c"}}))
	m.Focus()
	m.input.SetValue(">")
	m.cursor = 1

	// Send a message that the textinput doesn't consume as text (e.g.,
	// a window-size change). Cursor must not reset.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.cursor != 1 {
		t.Errorf("cursor = %d after non-input msg, want 1 (unchanged)", m.cursor)
	}
}

func TestCursorNavigation(t *testing.T) {
	newPaletteIn := func(t *testing.T) Model {
		t.Helper()
		m := New(WithCommands([]Item{
			Command{Name: "Open file"},
			Command{Name: "Save"},
			Command{Name: "Quit"},
		}))
		m.Focus()
		m.input.SetValue(">")
		return m
	}

	tests := []struct {
		name string
		seed int
		key  tea.KeyPressMsg
		want int
	}{
		{"down from top", 0, arrowKey(tea.KeyDown), 1},
		{"down again", 1, arrowKey(tea.KeyDown), 2},
		{"down wraps from last", 2, arrowKey(tea.KeyDown), 0},
		{"up from middle", 1, arrowKey(tea.KeyUp), 0},
		{"up wraps from first", 0, arrowKey(tea.KeyUp), 2},
		{"ctrl+n is down", 0, ctrlKey('n'), 1},
		{"ctrl+p is up", 1, ctrlKey('p'), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newPaletteIn(t)
			m.cursor = tc.seed
			m, _ = m.Update(tc.key)
			if m.cursor != tc.want {
				t.Errorf("cursor = %d, want %d", m.cursor, tc.want)
			}
		})
	}
}

func TestNavigationDoesNotAlterInput(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "a"}, Command{Name: "b"}}))
	m.Focus()
	m.input.SetValue(">")

	m, _ = m.Update(arrowKey(tea.KeyDown))

	if v := m.input.Value(); v != ">" {
		t.Errorf("input mutated by navigation key: got %q, want %q", v, ">")
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

// runMarker is what a test Command's Run() emits — lets us assert
// that Run actually fired when Enter dispatches.
type runMarker struct{ id string }

func TestEnterDispatchesCommandRun(t *testing.T) {
	cmd := Command{
		Name: "Open file",
		Run:  func() tea.Cmd { return func() tea.Msg { return runMarker{id: "open"} } },
	}

	m := New(WithCommands([]Item{cmd}))
	m.Focus()
	m.input.SetValue(">")

	m, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd, expected dispatch")
	}

	// The batched cmd produces a BatchMsg containing the SelectedMsg
	// emitter and the Command's Run cmd.
	batchMsg, ok := dispatched().(tea.BatchMsg)
	if !ok {
		t.Fatalf("dispatched msg = %T, want tea.BatchMsg", dispatched())
	}
	if len(batchMsg) != 2 {
		t.Fatalf("BatchMsg holds %d cmds, want 2", len(batchMsg))
	}

	var sawSelected, sawRun bool
	for _, c := range batchMsg {
		switch v := c().(type) {
		case SelectedMsg:
			if v.Item.FilterValue() != "Open file" {
				t.Errorf("SelectedMsg.Item = %q, want Open file", v.Item.FilterValue())
			}
			sawSelected = true
		case runMarker:
			if v.id != "open" {
				t.Errorf("runMarker id = %q, want open", v.id)
			}
			sawRun = true
		}
	}
	if !sawSelected {
		t.Error("batch missing SelectedMsg")
	}
	if !sawRun {
		t.Error("batch missing Command.Run output")
	}
}

func TestEnterEmitsSelectedMsgWhenRunNil(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "Save", Run: nil}}))
	m.Focus()
	m.input.SetValue(">")

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd")
	}
	msg := dispatched()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("dispatched = %T, want SelectedMsg", msg)
	}
	if sel.Item.FilterValue() != "Save" {
		t.Errorf("SelectedMsg.Item = %q, want Save", sel.Item.FilterValue())
	}
}

func TestEnterOnNonCommandItem(t *testing.T) {
	// SearchMode result that isn't a Command: Enter should still
	// emit SelectedMsg so the host knows what was picked.
	m := New()
	m.Focus()
	m.input.SetValue("query")
	m.results = []Item{testItem{name: "hit"}}

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched == nil {
		t.Fatal("Enter returned nil cmd")
	}
	sel, ok := dispatched().(SelectedMsg)
	if !ok {
		t.Fatalf("dispatched = %T, want SelectedMsg", dispatched())
	}
	if sel.Item.FilterValue() != "hit" {
		t.Errorf("SelectedMsg.Item = %q, want hit", sel.Item.FilterValue())
	}
}

func TestEnterWithNoSelection(t *testing.T) {
	// Empty items list → no cmd, no panic.
	m := New()
	m.Focus()
	m.input.SetValue("nothing")

	_, dispatched := m.Update(arrowKey(tea.KeyEnter))
	if dispatched != nil {
		t.Errorf("Enter with no selection should return nil cmd, got %v", dispatched())
	}
}

func TestEnterNotForwardedToInput(t *testing.T) {
	// Enter must be consumed; the textinput must not see it (which
	// in some configurations would otherwise submit/blur the input).
	m := New(WithCommands([]Item{Command{Name: "a"}}))
	m.Focus()
	m.input.SetValue(">a")

	m, _ = m.Update(arrowKey(tea.KeyEnter))
	if v := m.input.Value(); v != ">a" {
		t.Errorf("input mutated by Enter: got %q, want %q", v, ">a")
	}
}

func TestNavigationWithNoItems(t *testing.T) {
	// SearchMode + no results → no items. Navigation must be a no-op,
	// not panic or set cursor out of bounds.
	m := New()
	m.Focus()
	m.input.SetValue("query")
	m.cursor = 0

	m, _ = m.Update(arrowKey(tea.KeyDown))
	if m.cursor != 0 {
		t.Errorf("cursor moved with no items: got %d, want 0", m.cursor)
	}
}

// arrowKey returns a tea.KeyPressMsg for a special key like tea.KeyDown.
func arrowKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

// ctrlKey returns a tea.KeyPressMsg for ctrl+<rune>.
func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

// itemNames extracts FilterValues for readable test failures.
func itemNames(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.FilterValue()
	}
	return out
}

// typeKey returns a tea.KeyPressMsg corresponding to a single typed
// character, suitable for feeding through Model.Update.
func typeKey(t *testing.T, ch string) tea.KeyPressMsg {
	t.Helper()
	r := []rune(ch)
	if len(r) != 1 {
		t.Fatalf("typeKey expects a single rune, got %q", ch)
	}
	return tea.KeyPressMsg{Code: r[0], Text: ch}
}

func TestSelectedBounds(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "a"}, Command{Name: "b"}}))
	m.input.SetValue(">")

	m.cursor = 0
	if got := m.Selected(); got == nil || got.FilterValue() != "a" {
		t.Errorf("cursor=0 Selected() = %v, want a", got)
	}

	m.cursor = -1
	if got := m.Selected(); got != nil {
		t.Errorf("cursor=-1 Selected() = %v, want nil", got)
	}

	m.cursor = 99
	if got := m.Selected(); got != nil {
		t.Errorf("cursor=99 Selected() = %v, want nil", got)
	}
}

func TestReset(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "a"}}))
	m.input.SetValue("hello")
	m.results = []Item{testItem{name: "x"}}
	m.cursor = 3
	m.paginator.Page = 2
	m.loading = true

	m.Reset()

	if m.Value() != "" {
		t.Errorf("Value() = %q after Reset, want empty", m.Value())
	}
	if m.results != nil {
		t.Errorf("results = %v after Reset, want nil", m.results)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d after Reset, want 0", m.cursor)
	}
	if m.paginator.Page != 0 {
		t.Errorf("paginator.Page = %d after Reset, want 0", m.paginator.Page)
	}
	if m.loading {
		t.Error("loading = true after Reset, want false")
	}
}

func TestOptionsApply(t *testing.T) {
	cmds := []Item{Command{Name: "open"}}
	custom := KeyMap{}
	styles := Styles{}
	search := SearchFunc(func(string) tea.Cmd { return nil })

	m := New(
		WithCommands(cmds),
		WithSearch(search),
		WithKeyMap(custom),
		WithStyles(styles),
		WithPageSize(7),
		WithPaginatorType(paginator.Arabic),
	)

	if len(m.commands) != 1 {
		t.Errorf("commands not applied: %v", m.commands)
	}
	if m.search == nil {
		t.Error("search not applied")
	}
	if m.paginator.PerPage != 7 {
		t.Errorf("paginator.PerPage = %d, want 7", m.paginator.PerPage)
	}
	if m.paginator.Type != paginator.Arabic {
		t.Errorf("paginator.Type = %v, want Arabic", m.paginator.Type)
	}
}

func TestPageSizeZeroSkipsPaginatorPerPage(t *testing.T) {
	// pageSize=0 means auto-fit, which lands in a later milestone.
	// Until then, it must NOT clobber the paginator's default PerPage.
	m := New(WithPageSize(0))

	if m.pageSize != 0 {
		t.Errorf("pageSize = %d, want 0", m.pageSize)
	}
	if m.paginator.PerPage == 0 {
		t.Error("paginator.PerPage should retain its default when pageSize=0")
	}
}

func TestCommandImplementsDefaultItem(t *testing.T) {
	var _ DefaultItem = Command{Name: "x", Desc: "y"}
}

func TestShortHelpBindings(t *testing.T) {
	m := New()
	bindings := m.ShortHelp()
	if len(bindings) != 3 {
		t.Fatalf("ShortHelp returned %d bindings, want 3", len(bindings))
	}

	// First entry is the synthetic "navigate" combo (no matched keys,
	// just a help label).
	if got := bindings[0].Help(); got.Key != "↑↓" || got.Desc != "navigate" {
		t.Errorf("ShortHelp[0] help = %+v, want ↑↓/navigate", got)
	}
	if got := bindings[1].Help(); got.Desc != "execute" {
		t.Errorf("ShortHelp[1] desc = %q, want execute", got.Desc)
	}
	if got := bindings[2].Help(); got.Desc != "cancel" {
		t.Errorf("ShortHelp[2] desc = %q, want cancel", got.Desc)
	}
}

func TestFullHelpGroups(t *testing.T) {
	groups := New().FullHelp()
	if len(groups) != 3 {
		t.Fatalf("FullHelp groups = %d, want 3", len(groups))
	}
	// Each group should hold ≥1 binding with non-empty help text.
	for i, g := range groups {
		if len(g) == 0 {
			t.Errorf("FullHelp group %d is empty", i)
		}
		for j, b := range g {
			if b.Help().Desc == "" {
				t.Errorf("FullHelp[%d][%d] missing help desc", i, j)
			}
		}
	}
}

func TestViewIncludesHelp(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "open"}}))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})

	out := m.View()
	if !strings.Contains(out, "navigate") {
		t.Errorf("View() missing help text 'navigate', got:\n%s", out)
	}
	if !strings.Contains(out, "execute") {
		t.Errorf("View() missing help text 'execute', got:\n%s", out)
	}
}

func TestWithHelpFalseHidesHelp(t *testing.T) {
	m := New(WithCommands([]Item{Command{Name: "open"}}), WithHelp(false))
	m.input.SetValue(">")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})

	out := m.View()
	if strings.Contains(out, "navigate") {
		t.Errorf("View() rendered help despite WithHelp(false), got:\n%s", out)
	}
}

func TestViewRendersInputAndItems(t *testing.T) {
	m := New(WithCommands([]Item{
		Command{Name: "open", Desc: "open it"},
		Command{Name: "save"},
	}))
	m.input.SetValue(">")
	m.width = 40

	out := m.View()

	// Both command titles should appear in the rendered output.
	for _, want := range []string{"open", "save"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestViewWithNoItems(t *testing.T) {
	// SearchMode + empty results should render the input inside the
	// container but no items list under it.
	m := New(WithCommands([]Item{Command{Name: "open"}, Command{Name: "save"}}))
	m.input.SetValue("hello")

	out := m.View()
	if !strings.Contains(out, "hello") {
		t.Errorf("View() missing input value, got %q", out)
	}
	for _, name := range []string{"open", "save"} {
		if strings.Contains(out, name) {
			t.Errorf("View() should not render commands in SearchMode, but found %q in:\n%s", name, out)
		}
	}
}

func TestDefaultKeyMapHasBindings(t *testing.T) {
	km := DefaultKeyMap()
	bindings := []struct {
		name string
		keys []string
	}{
		{"Execute", km.Execute.Keys()},
		{"Cancel", km.Cancel.Keys()},
		{"Down", km.Down.Keys()},
		{"Up", km.Up.Keys()},
		{"NextPage", km.NextPage.Keys()},
		{"PrevPage", km.PrevPage.Keys()},
	}
	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("DefaultKeyMap.%s has no keys", b.name)
		}
	}
}
