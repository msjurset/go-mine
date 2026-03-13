package app

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// Autocomplete provides a dropdown suggestion list for text inputs.
// It is not a bubbletea Model itself — the owning view drives it.
type Autocomplete struct {
	suggestions []Suggestion
	visible     bool
	selected    int
	maxVisible  int
	corpus      []Suggestion // full list to filter from
	justAccepted bool        // suppress re-trigger until next real keystroke
}

// Suggestion is a single completable item.
type Suggestion struct {
	Text string // the completion text to insert
	Kind string // category label shown in the dropdown (e.g. "column", "keyword", "func")
}

func NewAutocomplete() Autocomplete {
	return Autocomplete{maxVisible: 8}
}

// SetCorpus replaces the full suggestion list.
func (a *Autocomplete) SetCorpus(corpus []Suggestion) {
	a.corpus = corpus
	a.suggestions = nil
	a.visible = false
}

// ClearJustAccepted resets the post-accept suppression. Call this when a
// real (non-accept) keystroke occurs so suggestions can resume.
func (a *Autocomplete) ClearJustAccepted() {
	a.justAccepted = false
}

// Update recomputes visible suggestions based on the current input text
// and cursor position. Returns true if the dropdown should be shown.
func (a *Autocomplete) Update(fullText string, cursorPos int) bool {
	// After accepting a suggestion, suppress until the next real keystroke
	if a.justAccepted {
		a.visible = false
		a.suggestions = nil
		return false
	}

	// Suppress inside quoted strings
	if insideQuotes(fullText, cursorPos) {
		a.visible = false
		a.suggestions = nil
		return false
	}

	prefix := extractToken(fullText, cursorPos)
	if prefix == "" {
		a.visible = false
		a.suggestions = nil
		return false
	}

	// Determine which suggestion kinds are excluded based on context
	excluded := excludedKinds(fullText, cursorPos)

	lower := strings.ToLower(prefix)

	var matches []Suggestion
	for _, s := range a.corpus {
		if excluded[s.Kind] {
			continue
		}
		sLower := strings.ToLower(s.Text)
		if strings.HasPrefix(sLower, lower) {
			matches = append(matches, s)
		}
	}

	// Also do substring matching for items not caught by prefix
	if len(matches) < a.maxVisible {
		seen := make(map[string]bool, len(matches))
		for _, m := range matches {
			seen[m.Text] = true
		}
		for _, s := range a.corpus {
			if excluded[s.Kind] {
				continue
			}
			sLower := strings.ToLower(s.Text)
			if !seen[s.Text] && strings.Contains(sLower, lower) {
				matches = append(matches, s)
				if len(matches) >= a.maxVisible*2 {
					break
				}
			}
		}
	}

	a.suggestions = matches
	a.visible = len(matches) > 0
	if a.selected >= len(matches) {
		a.selected = 0
	}
	return a.visible
}

// insideQuotes returns true if the cursor is between an opening and closing quote.
func insideQuotes(text string, cursorPos int) bool {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}
	inSingle := false
	inDouble := false
	for i := 0; i < cursorPos; i++ {
		ch := text[i]
		if ch == '"' && !inSingle {
			// Skip escaped quotes
			if i > 0 && text[i-1] == '\\' {
				continue
			}
			inDouble = !inDouble
		} else if ch == '\'' && !inDouble {
			if i > 0 && text[i-1] == '\\' {
				continue
			}
			inSingle = !inSingle
		}
	}
	return inSingle || inDouble
}

// excludedKinds determines which suggestion kinds should be suppressed
// based on the text context before the cursor.
func excludedKinds(text string, cursorPos int) map[string]bool {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}

	// Look at what precedes the current token (skip back past the token itself)
	tokenStart, _ := tokenBounds(text, cursorPos)
	before := strings.TrimRight(text[:tokenStart], " \t")

	// After a comparison operator, we're in value position — suppress
	// columns, methods, keywords, logic, funcs (user is typing a literal value)
	operators := []string{">=", "<=", "!=", "==", ">", "<", "LIKE ", "BETWEEN "}
	for _, op := range operators {
		if strings.HasSuffix(strings.ToUpper(before), op) || strings.HasSuffix(before, op) {
			return map[string]bool{
				"column":  true,
				"method":  true,
				"keyword": true,
				"logic":   true,
				"func":    true,
				"table":   true,
				"field":   true,
			}
		}
	}

	return nil
}

// Next moves selection down.
func (a *Autocomplete) Next() {
	if len(a.suggestions) == 0 {
		return
	}
	a.selected = (a.selected + 1) % len(a.suggestions)
}

// Prev moves selection up.
func (a *Autocomplete) Prev() {
	if len(a.suggestions) == 0 {
		return
	}
	a.selected = (a.selected - 1 + len(a.suggestions)) % len(a.suggestions)
}

// SelectedIsExact returns true if the currently selected suggestion exactly
// matches the token already typed. In that case, accepting would be a no-op
// and enter should fall through to execute the query instead.
func (a *Autocomplete) SelectedIsExact(fullText string, cursorPos int) bool {
	if !a.visible || len(a.suggestions) == 0 {
		return false
	}
	token := extractToken(fullText, cursorPos)
	return strings.EqualFold(a.suggestions[a.selected].Text, token)
}

// Accept returns the completed text with the current suggestion replacing
// the token at the cursor. Returns the new full text and new cursor position.
func (a *Autocomplete) Accept(fullText string, cursorPos int) (string, int) {
	if !a.visible || len(a.suggestions) == 0 {
		return fullText, cursorPos
	}

	chosen := a.suggestions[a.selected].Text
	tokenStart, tokenEnd := tokenBounds(fullText, cursorPos)

	newText := fullText[:tokenStart] + chosen + fullText[tokenEnd:]
	newCursor := tokenStart + len(chosen)

	a.visible = false
	a.suggestions = nil
	a.justAccepted = true
	return newText, newCursor
}

// Visible returns whether the dropdown is showing.
func (a *Autocomplete) Visible() bool {
	return a.visible
}

// Dismiss hides the dropdown.
func (a *Autocomplete) Dismiss() {
	a.visible = false
}

// View renders the dropdown as a styled string.
func (a *Autocomplete) View() string {
	if !a.visible || len(a.suggestions) == 0 {
		return ""
	}

	n := len(a.suggestions)
	if n > a.maxVisible {
		n = a.maxVisible
	}

	// Calculate scroll window around selected item
	start := 0
	if a.selected >= n {
		start = a.selected - n + 1
	}
	end := start + n
	if end > len(a.suggestions) {
		end = len(a.suggestions)
		start = end - n
		if start < 0 {
			start = 0
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		s := a.suggestions[i]
		text := s.Text
		if len(text) > 30 {
			text = text[:27] + "..."
		}
		kind := s.Kind
		if len(kind) > 8 {
			kind = kind[:8]
		}

		entry := lipgloss.NewStyle().Width(32).Render(
			"  " + text + strings.Repeat(" ", max(1, 30-len(text))) + kind,
		)

		if i == a.selected {
			entry = acSelectedStyle.Width(40).Render(
				"▸ " + text + strings.Repeat(" ", max(1, 30-len(text))) + kind,
			)
		} else {
			entry = acItemStyle.Width(40).Render(
				"  " + text + strings.Repeat(" ", max(1, 30-len(text))) + kind,
			)
		}
		lines = append(lines, entry)
	}

	if len(a.suggestions) > n {
		scrollInfo := acScrollStyle.Render(
			fmt.Sprintf("  ↑↓ %d more", len(a.suggestions)-n),
		)
		lines = append(lines, scrollInfo)
	}

	return acBorderStyle.Render(strings.Join(lines, "\n"))
}

// extractToken finds the word-like token ending at (or just before) cursorPos.
func extractToken(text string, cursorPos int) string {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}

	// Walk backwards from cursor to find token start
	start := cursorPos
	for start > 0 {
		r := rune(text[start-1])
		if isTokenChar(r) {
			start--
		} else {
			break
		}
	}

	token := text[start:cursorPos]
	// Only suggest if there's at least 1 char typed
	if len(token) < 1 {
		return ""
	}
	return token
}

// tokenBounds returns the start and end indices of the token at cursorPos.
func tokenBounds(text string, cursorPos int) (int, int) {
	if cursorPos > len(text) {
		cursorPos = len(text)
	}

	start := cursorPos
	for start > 0 && isTokenChar(rune(text[start-1])) {
		start--
	}

	end := cursorPos
	for end < len(text) && isTokenChar(rune(text[end])) {
		end++
	}

	return start, end
}

func isTokenChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

// --- Corpus builders for each view ---

// BuildFilterCorpus creates suggestions for the filter view.
func BuildFilterCorpus(colNames []string, colTypes []string) []Suggestion {
	var corpus []Suggestion

	stringMethods := []struct {
		suffix string
		kind   string
	}{
		{`.contains("`, "method"},
		{`.matches("`, "method"},
		{`.startswith("`, "method"},
		{`.endswith("`, "method"},
	}

	nullMethods := []struct {
		suffix string
		kind   string
	}{
		{".is_null", "method"},
		{".is_not_null", "method"},
	}

	// Column names + type-appropriate compound entries
	for i, name := range colNames {
		typ := ""
		if i < len(colTypes) {
			typ = colTypes[i]
		}
		corpus = append(corpus, Suggestion{Text: name, Kind: typ})

		// Null checks apply to all types
		for _, m := range nullMethods {
			corpus = append(corpus, Suggestion{Text: name + m.suffix, Kind: m.kind})
		}

		switch typ {
		case "str":
			// String methods
			for _, m := range stringMethods {
				corpus = append(corpus, Suggestion{Text: name + m.suffix, Kind: m.kind})
			}
		case "bool":
			// Boolean-specific comparisons
			corpus = append(corpus,
				Suggestion{Text: name + " == true", Kind: "value"},
				Suggestion{Text: name + " == false", Kind: "value"},
			)
		// Numeric types: no extra methods needed — user types operators directly
		}
	}

	// Logic keywords
	corpus = append(corpus,
		Suggestion{Text: "AND", Kind: "logic"},
		Suggestion{Text: "OR", Kind: "logic"},
	)
	return corpus
}

// BuildSQLCorpus creates suggestions for the SQL view.
func BuildSQLCorpus(colNames []string, tableNames []string) []Suggestion {
	var corpus []Suggestion

	// Column names
	for _, name := range colNames {
		corpus = append(corpus, Suggestion{Text: name, Kind: "column"})
	}

	// Table names
	for _, name := range tableNames {
		corpus = append(corpus, Suggestion{Text: name, Kind: "table"})
	}

	// SQL keywords
	sqlKeywords := []string{
		"SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN",
		"ORDER BY", "GROUP BY", "HAVING", "LIMIT", "OFFSET",
		"JOIN", "LEFT JOIN", "RIGHT JOIN", "INNER JOIN",
		"ON", "AS", "DISTINCT", "COUNT", "SUM", "AVG", "MIN", "MAX",
		"LIKE", "BETWEEN", "IS NULL", "IS NOT NULL",
		"ASC", "DESC", "CASE", "WHEN", "THEN", "ELSE", "END",
		"CAST", "COALESCE", "NULLIF", "UNION", "EXCEPT", "INTERSECT",
	}
	for _, kw := range sqlKeywords {
		corpus = append(corpus, Suggestion{Text: kw, Kind: "keyword"})
	}

	return corpus
}

// BuildJQCorpus creates suggestions for the JQ view.
func BuildJQCorpus(colNames []string) []Suggestion {
	var corpus []Suggestion

	// Dot-prefixed field access
	for _, name := range colNames {
		corpus = append(corpus, Suggestion{Text: "." + name, Kind: "field"})
	}

	// Common jq builtins
	jqBuiltins := []Suggestion{
		{Text: "select(", Kind: "func"},
		{Text: "map(", Kind: "func"},
		{Text: "group_by(", Kind: "func"},
		{Text: "sort_by(", Kind: "func"},
		{Text: "unique_by(", Kind: "func"},
		{Text: "unique", Kind: "func"},
		{Text: "flatten", Kind: "func"},
		{Text: "length", Kind: "func"},
		{Text: "keys", Kind: "func"},
		{Text: "values", Kind: "func"},
		{Text: "has(", Kind: "func"},
		{Text: "in(", Kind: "func"},
		{Text: "contains(", Kind: "func"},
		{Text: "inside(", Kind: "func"},
		{Text: "to_entries", Kind: "func"},
		{Text: "from_entries", Kind: "func"},
		{Text: "with_entries(", Kind: "func"},
		{Text: "add", Kind: "func"},
		{Text: "any", Kind: "func"},
		{Text: "all", Kind: "func"},
		{Text: "min_by(", Kind: "func"},
		{Text: "max_by(", Kind: "func"},
		{Text: "tostring", Kind: "func"},
		{Text: "tonumber", Kind: "func"},
		{Text: "ascii_downcase", Kind: "func"},
		{Text: "ascii_upcase", Kind: "func"},
		{Text: "ltrimstr(", Kind: "func"},
		{Text: "rtrimstr(", Kind: "func"},
		{Text: "split(", Kind: "func"},
		{Text: "join(", Kind: "func"},
		{Text: "test(", Kind: "func"},
		{Text: "match(", Kind: "func"},
		{Text: "capture(", Kind: "func"},
		{Text: "type", Kind: "func"},
		{Text: "empty", Kind: "func"},
		{Text: "error", Kind: "func"},
		{Text: "null", Kind: "value"},
		{Text: "true", Kind: "value"},
		{Text: "false", Kind: "value"},
		{Text: "not", Kind: "logic"},
		{Text: "and", Kind: "logic"},
		{Text: "or", Kind: "logic"},
		{Text: "if", Kind: "logic"},
		{Text: "then", Kind: "logic"},
		{Text: "elif", Kind: "logic"},
		{Text: "else", Kind: "logic"},
		{Text: "end", Kind: "logic"},
		{Text: "try", Kind: "logic"},
		{Text: "catch", Kind: "logic"},
		{Text: "reduce", Kind: "func"},
		{Text: "foreach", Kind: "func"},
		{Text: "limit(", Kind: "func"},
		{Text: "first(", Kind: "func"},
		{Text: "last(", Kind: "func"},
		{Text: "nth(", Kind: "func"},
		{Text: "range(", Kind: "func"},
		{Text: "indices(", Kind: "func"},
		{Text: "input", Kind: "func"},
		{Text: "inputs", Kind: "func"},
		{Text: "debug", Kind: "func"},
		{Text: "env", Kind: "func"},
		{Text: "path(", Kind: "func"},
		{Text: "getpath(", Kind: "func"},
		{Text: "setpath(", Kind: "func"},
		{Text: "delpaths(", Kind: "func"},
		{Text: "leaf_paths", Kind: "func"},
		{Text: "recurse", Kind: "func"},
		{Text: "walk(", Kind: "func"},
		{Text: "transpose", Kind: "func"},
		{Text: "ascii", Kind: "func"},
		{Text: "explode", Kind: "func"},
		{Text: "implode", Kind: "func"},
		{Text: "tojson", Kind: "func"},
		{Text: "fromjson", Kind: "func"},
		{Text: "infinite", Kind: "func"},
		{Text: "nan", Kind: "func"},
		{Text: "isinfinite", Kind: "func"},
		{Text: "isnan", Kind: "func"},
		{Text: "isnormal", Kind: "func"},
		{Text: "builtins", Kind: "func"},
	}
	corpus = append(corpus, jqBuiltins...)

	return corpus
}

