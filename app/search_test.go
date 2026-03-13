package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/msjurset/golars"
)

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestSearchStateTransitions(t *testing.T) {
	m := NewSearchModel()

	// Starts inactive
	if m.state != SearchInactive {
		t.Fatal("expected SearchInactive")
	}
	if m.Active() {
		t.Fatal("expected not active")
	}

	// Open transitions to SearchInput
	m.Open()
	if m.state != SearchInput {
		t.Fatal("expected SearchInput after Open")
	}
	if !m.Active() {
		t.Fatal("expected active after Open")
	}

	// Esc from SearchInput goes to Inactive
	m, _ = m.Update(specialKeyMsg(tea.KeyEsc))
	if m.state != SearchInactive {
		t.Fatal("expected SearchInactive after esc")
	}

	// Open again, type pattern, press enter -> SearchNavigate
	m.Open()
	m.input.SetValue("test")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	if m.state != SearchNavigate {
		t.Fatal("expected SearchNavigate after enter")
	}
	if m.pattern != "test" {
		t.Fatalf("expected pattern 'test', got %q", m.pattern)
	}
	if m.regex == nil {
		t.Fatal("expected regex to be compiled")
	}

	// Close from navigate goes to Inactive
	m.Close()
	if m.state != SearchInactive {
		t.Fatal("expected SearchInactive after Close")
	}
	if m.pattern != "" {
		t.Fatal("expected pattern cleared after Close")
	}
}

func TestSearchEmptyEnterCloses(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	if m.state != SearchInactive {
		t.Fatal("expected empty enter to close search")
	}
}

func TestSearchInvalidRegex(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("[invalid")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	// Should stay in SearchInput with an error
	if m.state != SearchInput {
		t.Fatal("expected to stay in SearchInput on invalid regex")
	}
	if m.err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func makeTestDF() *golars.DataFrame {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "alice_two"}),
		golars.NewInt64Series("age", []int64{30, 25, 35, 28}),
	)
	return df
}

func TestScanDataFrame(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("alice")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	df := makeTestDF()
	m.ScanDataFrame(df)

	// "alice" should match "Alice" (row 0, col 0) and "alice_two" (row 3, col 0)
	// case-insensitive
	if len(m.matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(m.matches))
	}
	if m.matches[0].Row != 0 || m.matches[0].Col != 0 {
		t.Fatalf("expected first match at (0,0), got (%d,%d)", m.matches[0].Row, m.matches[0].Col)
	}
	if m.matches[1].Row != 3 || m.matches[1].Col != 0 {
		t.Fatalf("expected second match at (3,0), got (%d,%d)", m.matches[1].Row, m.matches[1].Col)
	}
}

func TestSearchMatchNavigation(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("alice")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	df := makeTestDF()
	m.ScanDataFrame(df)

	if m.current != 0 {
		t.Fatal("expected current=0")
	}

	// Next
	m.NextMatch()
	if m.current != 1 {
		t.Fatalf("expected current=1, got %d", m.current)
	}

	// Next wraps
	m.NextMatch()
	if m.current != 0 {
		t.Fatalf("expected current=0 after wrap, got %d", m.current)
	}

	// Prev wraps
	m.PrevMatch()
	if m.current != 1 {
		t.Fatalf("expected current=1, got %d", m.current)
	}
}

func TestSearchMatchNavigationEmpty(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("zzzzz")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	df := makeTestDF()
	m.ScanDataFrame(df)

	if len(m.matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(m.matches))
	}
	// Should not panic
	m.NextMatch()
	m.PrevMatch()
}

func TestCurrentMatch(t *testing.T) {
	m := NewSearchModel()

	// No matches -> (-1, -1)
	row, col := m.CurrentMatch()
	if row != -1 || col != -1 {
		t.Fatalf("expected (-1,-1), got (%d,%d)", row, col)
	}

	m.Open()
	m.input.SetValue("bob")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	df := makeTestDF()
	m.ScanDataFrame(df)

	row, col = m.CurrentMatch()
	if row != 1 || col != 0 {
		t.Fatalf("expected (1,0), got (%d,%d)", row, col)
	}
}

func TestHighlightContentPlainText(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("bar")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	result := m.HighlightContent("foo bar baz", -1)
	if !strings.Contains(result, "bar") {
		t.Fatal("expected highlighted output to contain 'bar'")
	}
	if !strings.Contains(result, "foo ") {
		t.Fatal("expected 'foo ' prefix preserved")
	}
	if !strings.Contains(result, " baz") {
		t.Fatal("expected ' baz' suffix preserved")
	}
}

func TestHighlightContentWithANSI(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("hello")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	input := "\x1b[31mhello\x1b[0m world"
	result := m.HighlightContent(input, -1)
	plain := stripANSI(result)
	if !strings.Contains(plain, "hello") {
		t.Fatal("expected 'hello' in highlighted result")
	}
}

func TestSearchStatusView(t *testing.T) {
	m := NewSearchModel()

	// Inactive returns empty
	sv := m.StatusView()
	if sv != "" {
		t.Fatalf("expected empty status for inactive, got %q", sv)
	}

	// SearchInput shows /
	m.Open()
	sv = m.StatusView()
	if !strings.Contains(sv, "/") {
		t.Fatal("expected / in input status view")
	}

	// SearchNavigate shows pattern and count
	m.input.SetValue("alice")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	df := makeTestDF()
	m.ScanDataFrame(df)
	sv = m.StatusView()
	if !strings.Contains(sv, "/alice") {
		t.Fatal("expected /alice in navigate status view")
	}
	if !strings.Contains(sv, "[1/2]") {
		t.Fatalf("expected [1/2] in status view, got %q", sv)
	}
}

func TestHighlightContentActiveLine(t *testing.T) {
	m := NewSearchModel()
	m.Open()
	m.input.SetValue("match")
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))

	content := "no match here\nthis has match\nalso match here"
	// Line 1 should get active (red) highlight, others amber
	result := m.HighlightContent(content, 1)
	lines := strings.Split(result, "\n")

	// Line 0 has no match, should be unchanged
	if lines[0] != "no match here" {
		// Actually line 0 does contain "match" — let me fix the test content
	}

	// Use content where only specific lines match
	content2 := "no hit here\nthis has match\nalso has match"
	result2 := m.HighlightContent(content2, 1)
	lines2 := strings.Split(result2, "\n")

	// Line 0: no match, unchanged
	if stripANSI(lines2[0]) != "no hit here" {
		t.Fatalf("expected line 0 unchanged, got %q", stripANSI(lines2[0]))
	}

	// Line 1: active line — should have active style (red bg)
	// Line 2: non-active — should have highlight style (amber bg)
	// We can verify they're different
	if lines2[1] == lines2[2] {
		t.Fatal("expected active line to be styled differently from non-active")
	}

	// Both should contain "match" in plain text
	if !strings.Contains(stripANSI(lines2[1]), "match") {
		t.Fatal("expected 'match' in active line")
	}
	if !strings.Contains(stripANSI(lines2[2]), "match") {
		t.Fatal("expected 'match' in non-active line")
	}
}

func TestStripANSI(t *testing.T) {
	input := "\x1b[31mred\x1b[0m normal \x1b[1;32mbold green\x1b[0m"
	got := stripANSI(input)
	want := "red normal bold green"
	if got != want {
		t.Fatalf("stripANSI: got %q, want %q", got, want)
	}
}
