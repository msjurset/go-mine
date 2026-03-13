package app

import (
	"strings"
	"testing"

	"github.com/msjurset/golars"
)

func TestNewJQModel(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{30, 25}),
	)

	m := NewJQModel(df)
	if m.df != df {
		t.Error("expected df to be set")
	}
	if m.histIdx != -1 {
		t.Error("expected histIdx to be -1")
	}
}

func TestJQModelSetDataFrame(t *testing.T) {
	df1, _ := golars.NewDataFrame(
		golars.NewStringSeries("a", []string{"x"}),
	)
	df2, _ := golars.NewDataFrame(
		golars.NewStringSeries("b", []string{"y"}),
	)

	m := NewJQModel(df1)
	m.jsonCache = dataFrameToJSON(df1) // simulate cached state

	m.SetDataFrame(df2)
	if m.df != df2 {
		t.Error("expected df to be updated")
	}
	if m.jsonCache != nil {
		t.Error("expected jsonCache to be cleared")
	}
}

func TestJQModelExecuteTabular(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
		golars.NewInt64Series("age", []int64{30, 25, 35}),
	)

	m := NewJQModel(df)
	m.width = 80
	m.height = 40
	m.input.SetValue(`.[] | select(.age > 28)`)
	m.input.Focus()

	m, _ = m.executeQuery()

	if m.err != nil {
		t.Fatalf("unexpected error: %v", m.err)
	}
	if m.result == nil {
		t.Fatal("expected tabular result")
	}

	rows, _ := m.result.Shape()
	if rows != 2 {
		t.Errorf("expected 2 rows (Alice and Charlie), got %d", rows)
	}
}

func TestJQModelExecuteNonTabular(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice", "Bob"}),
		golars.NewInt64Series("age", []int64{30, 25}),
	)

	m := NewJQModel(df)
	m.width = 80
	m.height = 40
	m.input.SetValue(`[.[] | .name]`)
	m.input.Focus()

	m, _ = m.executeQuery()

	if m.err != nil {
		t.Fatalf("unexpected error: %v", m.err)
	}
	if m.result != nil {
		t.Error("expected non-tabular result (treeView)")
	}
	if !m.treeView.HasData() {
		t.Error("expected treeView to have data")
	}
	// Verify the view renders with the data
	m.treeView.SetSize(80, 40)
	view := m.treeView.View()
	if !strings.Contains(view, "Alice") {
		t.Error("expected treeView to contain Alice")
	}
}

func TestJQModelExecuteParseError(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("x", []string{"a"}),
	)

	m := NewJQModel(df)
	m.input.SetValue(`.[invalid`)
	m.input.Focus()

	m, _ = m.executeQuery()

	if m.err == nil {
		t.Error("expected parse error")
	}
}

func TestJQModelHistory(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("x", []string{"a"}),
	)

	m := NewJQModel(df)
	m.width = 80
	m.height = 40

	m.input.SetValue(`.[]`)
	m.input.Focus()
	m, _ = m.executeQuery()

	m.input.SetValue(`.[0]`)
	m.input.Focus()
	m, _ = m.executeQuery()

	if len(m.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(m.history))
	}
	if m.history[0] != ".[]" || m.history[1] != ".[0]" {
		t.Errorf("unexpected history: %v", m.history)
	}
}

func TestJQModelViewNoResults(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice"}),
	)

	m := NewJQModel(df)
	m.width = 80
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "JQ Query") {
		t.Error("expected view to contain header")
	}
	if !strings.Contains(view, "Example queries") {
		t.Error("expected view to show examples")
	}
	if !strings.Contains(view, "Schema Reference") {
		t.Error("expected view to show schema")
	}
}

func TestJQModelEmptyQuery(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("x", []string{"a"}),
	)

	m := NewJQModel(df)
	m.input.SetValue("")
	m.input.Focus()

	m, _ = m.executeQuery()
	if m.err != nil {
		t.Error("empty query should not produce an error")
	}
}

func TestJSONTreeView(t *testing.T) {
	results := []interface{}{"hello", "world"}
	tv := NewJSONTreeView()
	tv.SetSize(80, 40)
	tv.SetData(results, false)
	if !tv.HasData() {
		t.Error("expected tree view to have data")
	}
	view := tv.View()
	if !strings.Contains(view, "hello") {
		t.Error("expected output to contain hello")
	}
}

func TestJSONTreeViewTruncated(t *testing.T) {
	results := []interface{}{"hello", "world"}
	tv := NewJSONTreeView()
	tv.SetSize(80, 40)
	tv.SetData(results, true)
	view := tv.View()
	if !strings.Contains(view, "truncated") {
		t.Error("expected truncated message")
	}
}

func TestJSONTreeViewSingle(t *testing.T) {
	results := []interface{}{"solo"}
	tv := NewJSONTreeView()
	tv.SetSize(80, 40)
	tv.SetData(results, false)
	view := tv.View()
	// Single result should render as a string, not array
	if strings.Contains(view, "[") && !strings.Contains(view, "line") {
		t.Error("single result should not be wrapped in array brackets")
	}
}

func TestJSONTreeViewExpandCollapse(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"nested": map[string]interface{}{
			"a": 1,
			"b": 2,
		},
	}
	results := []interface{}{data}
	tv := NewJSONTreeView()
	tv.SetSize(80, 40)
	tv.SetData(results, false)

	// Should start expanded
	view1 := tv.View()
	lineCount1 := tv.LineCount()

	// Collapse all
	tv.CollapseAll()
	view2 := tv.View()
	lineCount2 := tv.LineCount()

	if lineCount2 >= lineCount1 {
		t.Errorf("collapsed should have fewer lines: %d >= %d", lineCount2, lineCount1)
	}

	// Expand all
	tv.ExpandAll()
	view3 := tv.View()
	lineCount3 := tv.LineCount()

	if lineCount3 != lineCount1 {
		t.Errorf("re-expanded should match original: %d != %d", lineCount3, lineCount1)
	}

	_ = view1
	_ = view2
	_ = view3
}

func TestJQModelAutocomplete(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewStringSeries("name", []string{"Alice"}),
		golars.NewInt64Series("age", []int64{30}),
	)

	m := NewJQModel(df)
	m.input.Focus()
	m.input.SetValue("sel")
	m.ac.Update("sel", 3)

	if !m.ac.Visible() {
		t.Error("expected autocomplete to be visible for 'sel'")
	}
}
