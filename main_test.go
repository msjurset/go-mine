package main

import (
	"testing"

	"github.com/msjurset/golars"
)

func TestGenerateSampleData(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"single row", 1},
		{"small", 10},
		{"medium", 100},
		{"default size", 10000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := generateSampleData(tt.n)
			if df == nil {
				t.Fatal("generateSampleData returned nil")
			}
			h, w := df.Shape()
			if h != tt.n {
				t.Errorf("expected %d rows, got %d", tt.n, h)
			}
			if w != 20 {
				t.Errorf("expected 20 columns, got %d", w)
			}
		})
	}
}

func TestGenerateSampleDataSchema(t *testing.T) {
	df := generateSampleData(10)
	schema := df.Schema()

	expected := []struct {
		name  string
		dtype golars.DataType
	}{
		{"id", golars.Int64},
		{"name", golars.String},
		{"email", golars.String},
		{"age", golars.Int64},
		{"city", golars.String},
		{"country", golars.String},
		{"department", golars.String},
		{"level", golars.String},
		{"salary", golars.Float64},
		{"bonus", golars.Float64},
		{"product", golars.String},
		{"years_exp", golars.Int64},
		{"perf_score", golars.Float64},
		{"satisfaction", golars.Int64},
		{"projects", golars.Int64},
		{"remote", golars.Boolean},
		{"status", golars.String},
		{"education", golars.String},
		{"team_size", golars.Int64},
		{"overtime_hrs", golars.Float64},
	}

	if schema.Len() != len(expected) {
		t.Fatalf("expected %d fields, got %d", len(expected), schema.Len())
	}

	for i, exp := range expected {
		f := schema.Field(i)
		if f.Name != exp.name {
			t.Errorf("field %d: expected name %q, got %q", i, exp.name, f.Name)
		}
		if f.Dtype != exp.dtype {
			t.Errorf("field %d (%s): expected dtype %v, got %v", i, exp.name, exp.dtype, f.Dtype)
		}
	}
}

func TestGenerateSampleDataValues(t *testing.T) {
	df := generateSampleData(100)

	// IDs should start at 100000
	idCol, _ := df.Column("id")
	firstID, _ := idCol.GetInt64(0)
	if firstID != 100000 {
		t.Errorf("expected first ID 100000, got %d", firstID)
	}
	lastID, _ := idCol.GetInt64(99)
	if lastID != 100099 {
		t.Errorf("expected last ID 100099, got %d", lastID)
	}

	// Ages should be between 20 and 64
	ageCol, _ := df.Column("age")
	for i := 0; i < ageCol.Len(); i++ {
		v, _ := ageCol.GetInt64(i)
		if v < 20 || v > 64 {
			t.Errorf("row %d: age %d out of range [20, 64]", i, v)
		}
	}

	// Salaries should be positive
	salaryCol, _ := df.Column("salary")
	for i := 0; i < salaryCol.Len(); i++ {
		v, _ := salaryCol.GetFloat64(i)
		if v <= 0 {
			t.Errorf("row %d: salary %.2f should be positive", i, v)
		}
	}

	// years_exp should never exceed age - 20
	yearsCol, _ := df.Column("years_exp")
	for i := 0; i < yearsCol.Len(); i++ {
		age, _ := ageCol.GetInt64(i)
		years, _ := yearsCol.GetInt64(i)
		if years > age-20 {
			t.Errorf("row %d: years_exp %d exceeds age-20 (%d)", i, years, age-20)
		}
		if years < 0 {
			t.Errorf("row %d: years_exp %d is negative", i, years)
		}
	}
}

func TestPrintInfo(t *testing.T) {
	df := generateSampleData(5)
	// Should not panic
	printInfo(df, "test.csv")
}
