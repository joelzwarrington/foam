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

	m.input.SetValue("> foo")
	if got := m.Items(); len(got) != 2 || got[0].FilterValue() != "open" {
		t.Errorf("CommandMode Items() = %v, want commands", got)
	}
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
	// SearchMode + empty results should render the input only (no
	// trailing newlines, no blank lines).
	m := New()
	m.input.SetValue("hello")
	out := m.View()
	if strings.Count(out, "\n") != 0 {
		t.Errorf("View() with no items should not add newlines, got %q", out)
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
