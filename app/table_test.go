package app

import (
	"fmt"
	"testing"

	"github.com/msjurset/golars"
)

func testDataFrame(t *testing.T) *golars.DataFrame {
	t.Helper()
	df, err := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3, 4, 5}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}),
		golars.NewFloat64Series("score", []float64{95.5, 82.3, 78.0, 91.2, 88.7}),
		golars.NewBooleanSeries("active", []bool{true, false, true, true, false}),
		golars.NewStringSeriesWithValidity("note", []string{"good", "", "ok", "great", ""}, []bool{true, false, true, true, false}),
	)
	if err != nil {
		t.Fatalf("testDataFrame: %v", err)
	}
	return df
}

func TestNewTableModel(t *testing.T) {
	df := testDataFrame(t)
	tm := NewTableModel(df)

	if tm.page != 0 {
		t.Errorf("expected initial page 0, got %d", tm.page)
	}
	if tm.cursorRow != 0 {
		t.Errorf("expected initial cursor 0, got %d", tm.cursorRow)
	}
	if tm.colOffset != 0 {
		t.Errorf("expected initial colOffset 0, got %d", tm.colOffset)
	}
	if tm.sortCol != -1 {
		t.Errorf("expected initial sortCol -1, got %d", tm.sortCol)
	}
	if tm.showDetail {
		t.Error("expected showDetail false initially")
	}
}

func TestSetDataFrame(t *testing.T) {
	df := testDataFrame(t)
	tm := NewTableModel(df)
	tm.page = 3
	tm.cursorRow = 2
	tm.sortCol = 1

	df2, _ := golars.NewDataFrame(
		golars.NewInt64Series("x", []int64{10, 20}),
	)
	tm.SetDataFrame(df2)

	if tm.page != 0 {
		t.Errorf("expected page reset to 0, got %d", tm.page)
	}
	if tm.cursorRow != 0 {
		t.Errorf("expected cursor reset to 0, got %d", tm.cursorRow)
	}
	if tm.sortCol != -1 {
		t.Errorf("expected sortCol reset to -1, got %d", tm.sortCol)
	}
}

func TestTotalPages(t *testing.T) {
	tests := []struct {
		name     string
		rows     int
		pageSize int
		expected int
	}{
		{"empty", 0, 20, 1},
		{"one page", 10, 20, 1},
		{"exact pages", 40, 20, 2},
		{"partial page", 41, 20, 3},
		{"single row", 1, 20, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := make([]int64, tt.rows)
			for i := range ids {
				ids[i] = int64(i)
			}
			var df *golars.DataFrame
			if tt.rows == 0 {
				df, _ = golars.NewDataFrame(golars.NewInt64Series("id", []int64{}))
			} else {
				df, _ = golars.NewDataFrame(golars.NewInt64Series("id", ids))
			}
			tm := NewTableModel(df)
			tm.pageSize = tt.pageSize
			got := tm.totalPages()
			if got != tt.expected {
				t.Errorf("expected %d pages, got %d", tt.expected, got)
			}
		})
	}
}

func TestFormatCellValue(t *testing.T) {
	df := testDataFrame(t)

	tests := []struct {
		colName  string
		row      int
		wide     bool
		expected string
	}{
		{"id", 0, true, "1"},
		{"id", 4, true, "5"},
		{"name", 0, true, "Alice"},
		{"name", 2, true, "Charlie"},
		{"score", 0, true, "95.5"},
		{"score", 2, true, "78.0"},
		{"active", 0, true, "true"},
		{"active", 1, true, "false"},
		{"note", 1, true, "null"}, // null value
		{"note", 0, true, "good"},
		// Compact mode
		{"score", 0, false, "95.5"},
		{"score", 2, false, "78.0"},
	}
	for _, tt := range tests {
		label := fmt.Sprintf("%s_%s_wide=%v", tt.colName, tt.expected, tt.wide)
		t.Run(label, func(t *testing.T) {
			col, _ := df.Column(tt.colName)
			got := formatCellValue(col, tt.row, tt.wide)
			if got != tt.expected {
				t.Errorf("formatCellValue(%s, %d, %v) = %q, want %q", tt.colName, tt.row, tt.wide, got, tt.expected)
			}
		})
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		name     string
		v        float64
		wide     bool
		expected string
	}{
		{"wide whole", 78.0, true, "78.0"},
		{"wide zero", 0.0, true, "0.0"},
		{"wide decimal", 95.5, true, "95.5"},
		{"wide precise", 3.14159265, true, "3.14159265"},
		{"wide large", 1234567.89, true, "1234567.89"},
		{"wide very large", 1e16, true, "1.000000e+16"},
		{"wide small", 0.000001, true, "0.000001"},
		{"compact whole", 78.0, false, "78.0"},
		{"compact decimal", 95.5, false, "95.5"},
		{"compact large", 12345.6, false, "1.235e+04"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFloat(tt.v, tt.wide)
			if got != tt.expected {
				t.Errorf("formatFloat(%g, %v) = %q, want %q", tt.v, tt.wide, got, tt.expected)
			}
		})
	}
}

func TestShortTypeName(t *testing.T) {
	tests := []struct {
		dt       golars.DataType
		expected string
	}{
		{golars.Int8, "i8"},
		{golars.Int16, "i16"},
		{golars.Int32, "i32"},
		{golars.Int64, "i64"},
		{golars.UInt8, "u8"},
		{golars.UInt16, "u16"},
		{golars.UInt32, "u32"},
		{golars.UInt64, "u64"},
		{golars.Float32, "f32"},
		{golars.Float64, "f64"},
		{golars.Boolean, "bool"},
		{golars.String, "str"},
		{golars.Date, "date"},
		{golars.DateTime, "datetime"},
		{golars.Time, "time"},
		{golars.Duration, "dur"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := shortTypeName(tt.dt)
			if got != tt.expected {
				t.Errorf("shortTypeName(%v) = %q, want %q", tt.dt, got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"hello world", 3, "hel"},
		{"hello world", 1, "h"},
		{"hello world", 0, ""},
		{"", 5, ""},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain text", "hello", "hello"},
		{"with color", "\x1b[31mred\x1b[0m", "red"},
		{"with bold", "\x1b[1mbold\x1b[0m", "bold"},
		{"empty", "", ""},
		{"nested", "\x1b[1m\x1b[31mboldred\x1b[0m\x1b[0m", "boldred"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripAnsi(tt.input)
			if got != tt.expected {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPadOrTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"pad short", "hi", 5, "hi   "},
		{"exact", "hello", 5, "hello"},
		{"truncate", "hello world", 5, "hello"},
		{"zero width", "hello", 0, ""},
		{"box drawing", "──────", 3, "───"},
		{"box drawing pad", "──", 5, "──   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := padOrTruncate(tt.input, tt.width)
			if got != tt.expected {
				t.Errorf("padOrTruncate(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.expected)
			}
		})
	}
}

func TestSliceVisual(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected string
	}{
		{"from start", "hello", 0, "hello"},
		{"from middle", "hello", 2, "llo"},
		{"past end", "hello", 10, ""},
		{"at end", "hello", 5, ""},
		{"box drawing", "──────", 3, "───"},
		{"box drawing from start", "╭──╮", 0, "╭──╮"},
		{"box drawing skip corner", "╭──╮", 1, "──╮"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sliceVisual(tt.input, tt.start)
			if got != tt.expected {
				t.Errorf("sliceVisual(%q, %d) = %q, want %q", tt.input, tt.start, got, tt.expected)
			}
		})
	}
}

func TestCalcColWidth(t *testing.T) {
	s := golars.NewStringSeries("name", []string{"Alice", "Bob", "Christopher"})
	w := calcColWidth(s, "name", false)
	// Should be at least len("Christopher") + 2 = 13
	if w < 13 {
		t.Errorf("expected width >= 13, got %d", w)
	}
	// Should be at least len("name") + 4 = 8
	if w < 8 {
		t.Errorf("expected width >= 8, got %d", w)
	}
}
