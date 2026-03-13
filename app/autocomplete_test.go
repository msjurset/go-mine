package app

import (
	"testing"
)

func TestAutocompleteBasic(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
		{Text: "age", Kind: "column"},
		{Text: "salary", Kind: "column"},
		{Text: "select(", Kind: "func"},
		{Text: "sort_by(", Kind: "func"},
	})

	// Typing "na" should suggest "name"
	visible := ac.Update("na", 2)
	if !visible {
		t.Error("expected suggestions for 'na'")
	}

	newText, _ := ac.Accept("na", 2)
	if newText != "name" {
		t.Errorf("expected 'name', got '%s'", newText)
	}
}

func TestAutocompleteCaseInsensitive(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "SELECT", Kind: "keyword"},
		{Text: "salary", Kind: "column"},
	})

	visible := ac.Update("sel", 3)
	if !visible {
		t.Error("expected case-insensitive match")
	}
}

func TestAutocompleteNoMatch(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
	})

	visible := ac.Update("xyz", 3)
	if visible {
		t.Error("expected no suggestions for 'xyz'")
	}
}

func TestAutocompleteDismiss(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
	})

	ac.Update("na", 2)
	if !ac.Visible() {
		t.Error("expected visible")
	}

	ac.Dismiss()
	if ac.Visible() {
		t.Error("expected hidden after dismiss")
	}
}

func TestAutocompleteNavigation(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "alpha", Kind: "col"},
		{Text: "able", Kind: "col"},
		{Text: "apex", Kind: "col"},
	})

	ac.Update("a", 1)
	if ac.selected != 0 {
		t.Errorf("expected selected 0, got %d", ac.selected)
	}

	ac.Next()
	if ac.selected != 1 {
		t.Errorf("expected selected 1, got %d", ac.selected)
	}

	ac.Prev()
	if ac.selected != 0 {
		t.Errorf("expected selected 0, got %d", ac.selected)
	}
}

func TestAutocompleteMiddleOfText(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "salary", Kind: "column"},
		{Text: "status", Kind: "column"},
	})

	// Cursor in the middle of: "sal > 100"
	visible := ac.Update("sal > 100", 3)
	if !visible {
		t.Error("expected suggestions at cursor position 3")
	}

	newText, newCursor := ac.Accept("sal > 100", 3)
	if newText != "salary > 100" {
		t.Errorf("expected 'salary > 100', got '%s'", newText)
	}
	if newCursor != 6 {
		t.Errorf("expected cursor at 6, got %d", newCursor)
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		text   string
		cursor int
		want   string
	}{
		{"name", 4, "name"},
		{"age > 30", 3, "age"},
		{".[] | select(", 13, ""},      // cursor after ( - no token
		{".[] | select", 12, "select"},
		{"SELECT * FROM", 6, "SELECT"},
		{"", 0, ""},
		{"a b", 1, "a"},
	}

	for _, tt := range tests {
		got := extractToken(tt.text, tt.cursor)
		if got != tt.want {
			t.Errorf("extractToken(%q, %d) = %q, want %q", tt.text, tt.cursor, got, tt.want)
		}
	}
}

func TestBuildFilterCorpusContainsColumns(t *testing.T) {
	corpus := BuildFilterCorpus([]string{"name", "age"}, []string{"str", "i64"})
	found := false
	for _, s := range corpus {
		if s.Text == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected corpus to contain 'name'")
	}
}

func TestBuildSQLCorpusContainsKeywords(t *testing.T) {
	corpus := BuildSQLCorpus([]string{"id"}, []string{"data"})
	foundSelect := false
	foundData := false
	for _, s := range corpus {
		if s.Text == "SELECT" {
			foundSelect = true
		}
		if s.Text == "data" {
			foundData = true
		}
	}
	if !foundSelect {
		t.Error("expected corpus to contain SELECT")
	}
	if !foundData {
		t.Error("expected corpus to contain 'data' table")
	}
}

func TestInsideQuotes(t *testing.T) {
	tests := []struct {
		text   string
		cursor int
		want   bool
	}{
		{`name == "Alice"`, 15, false},  // after closing quote
		{`name == "Ali`, 12, true},      // inside double quotes
		{`name == "Alice"`, 10, true},   // inside double quotes
		{`name == 'Ali`, 12, true},      // inside single quotes
		{`name == "Alice" AND age`, 23, false}, // after quotes, new token
		{`name`, 4, false},              // no quotes at all
		{`name == ""`, 9, true},         // between empty quotes
		{`name == "Ali\"ce`, 16, true},  // escaped quote, still inside
	}

	for _, tt := range tests {
		got := insideQuotes(tt.text, tt.cursor)
		if got != tt.want {
			t.Errorf("insideQuotes(%q, %d) = %v, want %v", tt.text, tt.cursor, got, tt.want)
		}
	}
}

func TestSuppressInsideQuotes(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
		{Text: ".contains(\"", Kind: "method"},
		{Text: "AND", Kind: "logic"},
	})

	// Inside a quoted string value — should suppress all suggestions
	visible := ac.Update(`name == "na`, 11)
	if visible {
		t.Error("expected no suggestions inside quoted string")
	}
}

func TestSuppressAfterOperator(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
		{Text: "age", Kind: "column"},
		{Text: ".contains(\"", Kind: "method"},
		{Text: "AND", Kind: "logic"},
	})

	// After == operator, typing a value — columns/methods should be suppressed
	visible := ac.Update("name == na", 10)
	if visible {
		t.Error("expected no suggestions in value position after ==")
	}
}

func TestSuggestionsAfterAND(t *testing.T) {
	ac := NewAutocomplete()
	ac.SetCorpus([]Suggestion{
		{Text: "name", Kind: "column"},
		{Text: "age", Kind: "column"},
		{Text: "AND", Kind: "logic"},
	})

	// After AND, typing a new column name — should suggest columns
	visible := ac.Update("name == \"Alice\" AND ag", 22)
	if !visible {
		t.Error("expected suggestions after AND for column name")
	}
}

func TestBuildJQCorpusContainsFields(t *testing.T) {
	corpus := BuildJQCorpus([]string{"name", "age"})
	found := false
	for _, s := range corpus {
		if s.Text == ".name" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected corpus to contain '.name'")
	}
}
