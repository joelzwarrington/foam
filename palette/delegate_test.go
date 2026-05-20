package palette

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var updateGolden = flag.Bool("update", false, "regenerate golden files in testdata/")

type plainItem struct{ name string }

func (p plainItem) FilterValue() string { return p.name }

type fullItem struct {
	title string
	desc  string
}

func (f fullItem) FilterValue() string { return f.title }
func (f fullItem) Title() string       { return f.title }
func (f fullItem) Description() string { return f.desc }

func TestDefaultDelegateRender(t *testing.T) {
	cases := []struct {
		name   string
		item   Item
		index  int
		cursor int
		width  int
		setup  func(*DefaultDelegate)
	}{
		{
			name:   "selected_with_desc",
			item:   fullItem{title: "Open file", desc: "Open a file in the editor"},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "unselected_with_desc",
			item:   fullItem{title: "Open file", desc: "Open a file in the editor"},
			index:  1,
			cursor: 0,
			width:  40,
		},
		{
			name:   "selected_without_desc",
			item:   fullItem{title: "Save", desc: ""},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "narrow_truncated",
			item:   fullItem{title: "Open a very long command name", desc: "with a description that also overflows"},
			index:  0,
			cursor: 0,
			width:  16,
		},
		{
			name:   "wide_no_truncation",
			item:   fullItem{title: "Open file", desc: "Short desc"},
			index:  0,
			cursor: 0,
			width:  120,
		},
		{
			name:   "plain_item_filter_value",
			item:   plainItem{name: "raw-result"},
			index:  0,
			cursor: 0,
			width:  40,
		},
		{
			name:   "show_description_off",
			item:   fullItem{title: "Open file", desc: "hidden in this mode"},
			index:  0,
			cursor: 0,
			width:  40,
			setup: func(d *DefaultDelegate) {
				d.ShowDescription = false
			},
		},
		{
			name:   "unknown_width_no_truncation",
			item:   fullItem{title: "Open a very long command name", desc: "with a long description"},
			index:  0,
			cursor: 0,
			width:  0, // width unset: render full text, no ellipsis
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDefaultDelegate()
			if tc.setup != nil {
				tc.setup(&d)
			}
			m := New()
			m.cursor = tc.cursor
			m.width = tc.width

			var buf bytes.Buffer
			d.Render(&buf, m, tc.index, tc.item)

			// Golden files store plaintext (ANSI escape codes stripped) so
			// they remain readable in diffs and stable across lipgloss
			// styling tweaks. Layout is what we're locking in here.
			got := ansi.Strip(buf.String())

			path := filepath.Join("testdata", tc.name+".golden")
			if *updateGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatalf("mkdir testdata: %v", err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if got != string(want) {
				t.Errorf("render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}
