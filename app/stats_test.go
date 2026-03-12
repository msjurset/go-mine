package app

import (
	"testing"

	"github.com/msjurset/golars"
)

func TestIsNumeric(t *testing.T) {
	numericTypes := []golars.DataType{
		golars.Int8, golars.Int16, golars.Int32, golars.Int64,
		golars.UInt8, golars.UInt16, golars.UInt32, golars.UInt64,
		golars.Float32, golars.Float64,
	}
	for _, dt := range numericTypes {
		if !isNumeric(dt) {
			t.Errorf("isNumeric(%v) = false, want true", dt)
		}
	}

	nonNumericTypes := []golars.DataType{
		golars.String, golars.Boolean, golars.Date, golars.DateTime,
		golars.Time, golars.Duration,
	}
	for _, dt := range nonNumericTypes {
		if isNumeric(dt) {
			t.Errorf("isNumeric(%v) = true, want false", dt)
		}
	}
}

func TestRenderMiniTable(t *testing.T) {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("id", []int64{1, 2, 3}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie"}),
	)

	result := renderMiniTable(df, 80)
	if result == "" {
		t.Error("renderMiniTable returned empty string for valid DataFrame")
	}

	// Nil DataFrame
	result = renderMiniTable(nil, 80)
	if result != "" {
		t.Errorf("renderMiniTable(nil) = %q, want empty", result)
	}
}

func TestNewStatsModel(t *testing.T) {
	df := testDataFrame()
	sm := NewStatsModel(df)

	if sm.df != df {
		t.Error("NewStatsModel did not store DataFrame")
	}
	if sm.colIndex != 0 {
		t.Errorf("expected initial colIndex 0, got %d", sm.colIndex)
	}
}

func TestStatsModelSetDataFrame(t *testing.T) {
	df := testDataFrame()
	sm := NewStatsModel(df)
	sm.colIndex = 3
	sm.scrollY = 10

	df2, _ := golars.NewDataFrame(golars.NewInt64Series("x", []int64{1}))
	sm.SetDataFrame(df2)

	if sm.colIndex != 0 {
		t.Errorf("expected colIndex reset to 0, got %d", sm.colIndex)
	}
	if sm.scrollY != 0 {
		t.Errorf("expected scrollY reset to 0, got %d", sm.scrollY)
	}
}
