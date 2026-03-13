package app

import (
	"testing"

	"github.com/msjurset/golars"
)

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		dt   golars.DataType
		want bool
	}{
		{golars.Int8, true},
		{golars.Int16, true},
		{golars.Int32, true},
		{golars.Int64, true},
		{golars.UInt8, true},
		{golars.UInt16, true},
		{golars.UInt32, true},
		{golars.UInt64, true},
		{golars.Float32, true},
		{golars.Float64, true},
		{golars.String, false},
		{golars.Boolean, false},
		{golars.Date, false},
		{golars.DateTime, false},
		{golars.Time, false},
		{golars.Duration, false},
	}
	for _, tt := range tests {
		t.Run(shortTypeName(tt.dt), func(t *testing.T) {
			if got := isNumeric(tt.dt); got != tt.want {
				t.Errorf("isNumeric(%v) = %v, want %v", tt.dt, got, tt.want)
			}
		})
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
	df := testDataFrame(t)
	sm := NewStatsModel(df)

	if sm.df != df {
		t.Error("NewStatsModel did not store DataFrame")
	}
	if sm.colIndex != 0 {
		t.Errorf("expected initial colIndex 0, got %d", sm.colIndex)
	}
}

func TestStatsModelSetDataFrame(t *testing.T) {
	df := testDataFrame(t)
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
