package app

import (
	"testing"

	"github.com/msjurset/golars"
)

func TestSchemaFields(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2}),
		golars.NewStringSeries("name", []string{"a", "b"}),
		golars.NewFloat64Series("value", []float64{1.0, 2.0}),
	)

	fields := schemaFields(df.Schema())
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	if fields[0].Name != "id" {
		t.Errorf("field 0: expected 'id', got %q", fields[0].Name)
	}
	if fields[1].Name != "name" {
		t.Errorf("field 1: expected 'name', got %q", fields[1].Name)
	}
	if fields[2].Name != "value" {
		t.Errorf("field 2: expected 'value', got %q", fields[2].Name)
	}
}

func TestGetColumn(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"a", "b", "c"}),
	)

	col := getColumn(df, "id")
	if col == nil {
		t.Fatal("getColumn returned nil for existing column")
	}
	if col.Len() != 3 {
		t.Errorf("expected len 3, got %d", col.Len())
	}

	// Non-existent column returns nil
	col = getColumn(df, "nonexistent")
	if col != nil {
		t.Error("getColumn should return nil for non-existent column")
	}
}

func TestCleanFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"csv extension", "sales.csv", "sales"},
		{"parquet extension", "sales.parquet", "sales"},
		{"json extension", "data.json", "data"},
		{"tsv extension", "report.tsv", "report"},
		{"dashes to underscores", "my-data.csv", "my_data"},
		{"spaces to underscores", "my data.csv", "my_data"},
		{"no extension", "rawdata", "rawdata"},
		{"multiple replacements", "my-big data.parquet", "my_big_data"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanFileName(tt.input)
			if got != tt.expected {
				t.Errorf("cleanFileName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestClampedScroll(t *testing.T) {
	tests := []struct {
		name          string
		scrollY       int
		lineCount     int
		visibleHeight int
		wantStart     int
		wantEnd       int
	}{
		{"zero lines", 0, 0, 10, 0, 0},
		{"scroll past end", 100, 5, 10, 0, 5},
		{"exact fit", 0, 10, 10, 0, 10},
		{"normal scroll", 5, 20, 10, 5, 15},
		{"clamp to max", 15, 20, 10, 10, 20},
		{"visible exceeds lines", 0, 3, 10, 0, 3},
		{"single line", 0, 1, 10, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := clampedScroll(tt.scrollY, tt.lineCount, tt.visibleHeight)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("clampedScroll(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.scrollY, tt.lineCount, tt.visibleHeight,
					start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
