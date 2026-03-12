package app

import (
	"testing"

	"github.com/msjurset/golars"
)

func filterTestDF() *golars.DataFrame {
	df, _ := golars.NewDataFrame(
		golars.NewInt64Series("age", []int64{25, 30, 35, 40, 45}),
		golars.NewStringSeries("name", []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}),
		golars.NewFloat64Series("score", []float64{95.5, 82.3, 78.0, 91.2, 88.7}),
		golars.NewFloat64SeriesWithValidity("bonus", []float64{100, 0, 200, 300, 0}, []bool{true, false, true, true, false}),
		golars.NewBooleanSeries("active", []bool{true, false, true, true, false}),
	)
	return df
}

func TestParseFilterExprComparison(t *testing.T) {
	df := filterTestDF()

	tests := []struct {
		name     string
		expr     string
		wantRows int
	}{
		{"greater than", "age > 30", 3},
		{"greater equal", "age >= 35", 3},
		{"less than", "age < 35", 2},
		{"less equal", "age <= 35", 3},
		{"equal int", "age == 30", 1},
		{"not equal", "age != 30", 4},
		{"equal string", `name == "Alice"`, 1},
		{"float comparison", "score > 90.0", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parseFilterExpr(tt.expr, df)
			if err != nil {
				t.Fatalf("parseFilterExpr(%q) error: %v", tt.expr, err)
			}

			ctx := &golars.ExprContext{DF: df}
			mask, err := expr.Evaluate(ctx)
			if err != nil {
				t.Fatalf("evaluate error: %v", err)
			}

			filtered, err := df.Filter(mask)
			if err != nil {
				t.Fatalf("filter error: %v", err)
			}

			if filtered.Height() != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, filtered.Height())
			}
		})
	}
}

func TestParseFilterExprMethods(t *testing.T) {
	df := filterTestDF()

	tests := []struct {
		name     string
		expr     string
		wantRows int
	}{
		{"is_null", "bonus.is_null", 2},
		{"is_not_null", "bonus.is_not_null", 3},
		{"contains", `name.contains("li")`, 2},          // Alice, Charlie
		{"matches", `name.matches("^[A-C]")`, 3},        // Alice, Bob, Charlie
		{"startswith", `name.startswith("Al")`, 1},       // Alice
		{"endswith", `name.endswith("e")`, 3},            // Alice, Charlie, Eve
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parseFilterExpr(tt.expr, df)
			if err != nil {
				t.Fatalf("parseFilterExpr(%q) error: %v", tt.expr, err)
			}

			ctx := &golars.ExprContext{DF: df}
			mask, err := expr.Evaluate(ctx)
			if err != nil {
				t.Fatalf("evaluate error: %v", err)
			}

			filtered, err := df.Filter(mask)
			if err != nil {
				t.Fatalf("filter error: %v", err)
			}

			if filtered.Height() != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, filtered.Height())
			}
		})
	}
}

func TestParseFilterExprLogical(t *testing.T) {
	df := filterTestDF()

	tests := []struct {
		name     string
		expr     string
		wantRows int
	}{
		{"AND", "age > 30 AND score > 90.0", 1},    // Diana (40, 91.2)
		{"OR", "age == 25 OR age == 45", 2},         // Alice, Eve
		{"AND range", "age >= 30 AND age <= 40", 3}, // Bob, Charlie, Diana
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parseFilterExpr(tt.expr, df)
			if err != nil {
				t.Fatalf("parseFilterExpr(%q) error: %v", tt.expr, err)
			}

			ctx := &golars.ExprContext{DF: df}
			mask, err := expr.Evaluate(ctx)
			if err != nil {
				t.Fatalf("evaluate error: %v", err)
			}

			filtered, err := df.Filter(mask)
			if err != nil {
				t.Fatalf("filter error: %v", err)
			}

			if filtered.Height() != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, filtered.Height())
			}
		})
	}
}

func TestParseFilterExprErrors(t *testing.T) {
	df := filterTestDF()

	tests := []struct {
		name string
		expr string
	}{
		{"unknown column", "nonexistent > 5"},
		{"unparseable", "just some text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFilterExpr(tt.expr, df)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.expr)
			}
		})
	}
}

func TestSplitLogical(t *testing.T) {
	tests := []struct {
		input    string
		wantOK   bool
		wantOp   string
		wantLeft string
	}{
		{"a > 1 AND b < 2", true, "AND", "a > 1"},
		{"x == 1 OR y == 2", true, "OR", "x == 1"},
		{"no logical here", false, "", ""},
		{"a > 1 and b < 2", true, "AND", "a > 1"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parts, op, ok := splitLogical(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if op != tt.wantOp {
					t.Errorf("op = %q, want %q", op, tt.wantOp)
				}
				if parts[0] != tt.wantLeft {
					t.Errorf("left = %q, want %q", parts[0], tt.wantLeft)
				}
			}
		})
	}
}

func TestParseLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		dt    golars.DataType
	}{
		{"quoted string", `"hello"`, golars.String},
		{"single quoted", `'hello'`, golars.String},
		{"true", "true", golars.Boolean},
		{"false", "FALSE", golars.Boolean},
		{"integer", "42", golars.Int64},
		{"float", "3.14", golars.Float64},
		{"int as float", "42", golars.Float64},
		{"bare string", "hello", golars.String},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLiteral(tt.input, tt.dt)
			if result == nil {
				t.Errorf("parseLiteral(%q, %v) returned nil", tt.input, tt.dt)
			}
		})
	}
}
