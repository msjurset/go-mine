package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type FilterModel struct {
	df      *golars.DataFrame
	input   textinput.Model
	history []string
	histIdx int
	err     error
	preview *golars.DataFrame
	width   int
	height  int
}

func NewFilterModel(df *golars.DataFrame) FilterModel {
	ti := textinput.New()
	ti.Placeholder = `e.g. age > 30, name == "Alice", score >= 80.0 AND score <= 100.0`
	ti.CharLimit = 256
	ti.Width = 80

	return FilterModel{
		df:      df,
		input:   ti,
		histIdx: -1,
	}
}

func (m *FilterModel) Focus() {
	m.input.Focus()
}

func (m *FilterModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 20
}

func (m FilterModel) Update(msg tea.Msg) (FilterModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m.applyFilter()
		case "ctrl+r":
			m.err = nil
			m.preview = nil
			m.input.SetValue("")
			return m, func() tea.Msg { return FilterClearedMsg{} }
		case "esc":
			m.input.Blur()
			return m, nil
		case "up":
			if len(m.history) > 0 {
				if m.histIdx < len(m.history)-1 {
					m.histIdx++
				}
				m.input.SetValue(m.history[len(m.history)-1-m.histIdx])
			}
			return m, nil
		case "down":
			if m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[len(m.history)-1-m.histIdx])
			} else {
				m.histIdx = -1
				m.input.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m FilterModel) applyFilter() (FilterModel, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		m.err = nil
		m.preview = nil
		return m, func() tea.Msg { return FilterClearedMsg{} }
	}

	expr, err := parseFilterExpr(text, m.df)
	if err != nil {
		m.err = err
		m.preview = nil
		return m, nil
	}

	ctx := &golars.ExprContext{DF: m.df}
	mask, err := expr.Evaluate(ctx)
	if err != nil {
		m.err = fmt.Errorf("evaluate: %w", err)
		m.preview = nil
		return m, nil
	}

	filtered, err := m.df.Filter(mask)
	if err != nil {
		m.err = fmt.Errorf("filter: %w", err)
		m.preview = nil
		return m, nil
	}

	m.err = nil
	m.history = append(m.history, text)
	m.histIdx = -1
	m.preview = filtered.Head(5)

	resultDF := filtered
	return m, func() tea.Msg { return FilterAppliedMsg{DF: resultDF} }
}

func (m FilterModel) View() string {
	var b strings.Builder

	b.WriteString(statHeaderStyle.Render("Filter Data") + "\n\n")
	b.WriteString(promptStyle.Render("  Filter: ") + m.input.View() + "\n\n")

	// Help text
	b.WriteString(helpStyle.Render("  Syntax examples:") + "\n")
	b.WriteString(helpStyle.Render("    column > 42          column == \"text\"       column != 0") + "\n")
	b.WriteString(helpStyle.Render("    column >= 10 AND column <= 100             column.is_null") + "\n")
	b.WriteString(helpStyle.Render("    column.contains(\"substr\")                  column.is_not_null") + "\n")
	b.WriteString(helpStyle.Render("  ") + "\n")
	b.WriteString(helpStyle.Render("  enter:apply  ctrl+r:clear filter  ↑↓:history  esc:unfocus") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n\n")
	}

	if m.preview != nil && !m.preview.IsEmpty() {
		b.WriteString(successStyle.Render(fmt.Sprintf("  Preview (showing %d of matched rows):", m.preview.Height())) + "\n")
		b.WriteString(renderMiniTable(m.preview, m.width-4))
	}

	// Show available columns
	b.WriteString("\n\n" + statHeaderStyle.Render("Available Columns") + "\n")
	fields := schemaFields(m.df.Schema())
	var cols []string
	for _, f := range fields {
		cols = append(cols, fmt.Sprintf("  %s (%s)", f.Name, shortTypeName(f.Dtype)))
	}

	colLines := strings.Join(cols, "\n")
	maxLines := m.height - 18
	lines := strings.Split(colLines, "\n")
	if len(lines) > maxLines && maxLines > 0 {
		colLines = strings.Join(lines[:maxLines], "\n") + "\n  ..."
	}
	b.WriteString(helpStyle.Render(colLines))

	return lipgloss.NewStyle().Width(m.width).Render(b.String())
}

// parseFilterExpr parses a simple filter expression string into a golars Expr.
// Supports: col op value, col.method(arg), expr AND/OR expr
func parseFilterExpr(text string, df *golars.DataFrame) (golars.Expr, error) {
	text = strings.TrimSpace(text)

	// Handle AND/OR (split on first occurrence, respecting parentheses)
	if parts, op, ok := splitLogical(text); ok {
		left, err := parseFilterExpr(parts[0], df)
		if err != nil {
			return nil, err
		}
		right, err := parseFilterExpr(parts[1], df)
		if err != nil {
			return nil, err
		}
		if op == "AND" {
			return left.And(right), nil
		}
		return left.Or(right), nil
	}

	// Handle method calls: col.is_null, col.is_not_null, col.contains("...")
	if idx := strings.Index(text, ".is_null"); idx > 0 && !strings.Contains(text[:idx], " ") {
		colName := strings.TrimSpace(text[:idx])
		return golars.Col(colName).IsNull(), nil
	}
	if idx := strings.Index(text, ".is_not_null"); idx > 0 && !strings.Contains(text[:idx], " ") {
		colName := strings.TrimSpace(text[:idx])
		return golars.Col(colName).IsNotNull(), nil
	}
	if idx := strings.Index(text, ".contains("); idx > 0 && strings.HasSuffix(text, ")") {
		colName := strings.TrimSpace(text[:idx])
		arg := text[idx+len(".contains(") : len(text)-1]
		arg = strings.Trim(arg, `"'`)
		return golars.Col(colName).Str().Contains(arg), nil
	}

	// Handle comparison: col op value
	operators := []struct {
		sym string
		fn  func(golars.Expr, golars.Expr) golars.Expr
	}{
		{">=", func(a, b golars.Expr) golars.Expr { return a.Gte(b) }},
		{"<=", func(a, b golars.Expr) golars.Expr { return a.Lte(b) }},
		{"!=", func(a, b golars.Expr) golars.Expr { return a.Neq(b) }},
		{"==", func(a, b golars.Expr) golars.Expr { return a.Eq(b) }},
		{">", func(a, b golars.Expr) golars.Expr { return a.Gt(b) }},
		{"<", func(a, b golars.Expr) golars.Expr { return a.Lt(b) }},
	}

	for _, op := range operators {
		parts := strings.SplitN(text, op.sym, 2)
		if len(parts) == 2 {
			colName := strings.TrimSpace(parts[0])
			valStr := strings.TrimSpace(parts[1])

			// Validate column exists
			if col, err := df.Column(colName); err != nil || col == nil {
				return nil, fmt.Errorf("unknown column: %q", colName)
			}

			lit := parseLiteral(valStr, getColumn(df, colName).DataType())
			return op.fn(golars.Col(colName), lit), nil
		}
	}

	return nil, fmt.Errorf("cannot parse filter expression: %q\nExpected: column operator value (e.g. age > 30)", text)
}

func splitLogical(text string) ([2]string, string, bool) {
	// Look for AND or OR (case insensitive) at word boundaries
	for _, op := range []string{" AND ", " OR "} {
		idx := strings.Index(strings.ToUpper(text), op)
		if idx > 0 {
			return [2]string{text[:idx], text[idx+len(op):]}, strings.TrimSpace(strings.ToUpper(op)), true
		}
	}
	return [2]string{}, "", false
}

func parseLiteral(s string, dt golars.DataType) golars.Expr {
	s = strings.TrimSpace(s)

	// Quoted string
	if (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`)) {
		return golars.Lit(s[1 : len(s)-1])
	}

	// Boolean
	if strings.ToLower(s) == "true" {
		return golars.Lit(true)
	}
	if strings.ToLower(s) == "false" {
		return golars.Lit(false)
	}

	// Try integer
	if iv, err := strconv.ParseInt(s, 10, 64); err == nil {
		if isNumeric(dt) && (dt == golars.Float32 || dt == golars.Float64) {
			return golars.Lit(float64(iv))
		}
		return golars.Lit(iv)
	}

	// Try float
	if fv, err := strconv.ParseFloat(s, 64); err == nil {
		return golars.Lit(fv)
	}

	// Fall back to string
	return golars.Lit(s)
}
