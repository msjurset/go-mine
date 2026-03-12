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
