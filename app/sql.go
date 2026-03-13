package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type SQLModel struct {
	df      *golars.DataFrame
	sqlCtx  *golars.SQLContext
	input   textinput.Model
	history []string
	histIdx int
	result  *golars.DataFrame
	table   TableModel
	err     error
	width   int
	height  int
	ac      Autocomplete
}

func NewSQLModel(df *golars.DataFrame, fileName string) SQLModel {
	ti := textinput.New()
	ti.Placeholder = `SELECT * FROM data WHERE ... GROUP BY ... ORDER BY ... LIMIT 20`
	ti.CharLimit = 512
	ti.Width = 80

	ctx := golars.NewSQLContext()
	ctx.Register("data", df)

	cleanName := cleanFileName(fileName)
	if cleanName != "data" && cleanName != "" {
		ctx.Register(cleanName, df)
	}

	ac := NewAutocomplete()
	ac.SetCorpus(buildSQLCorpus(df, fileName))

	return SQLModel{
		df:      df,
		sqlCtx:  ctx,
		input:   ti,
		histIdx: -1,
		ac:      ac,
	}
}

func buildSQLCorpus(df *golars.DataFrame, fileName string) []Suggestion {
	fields := schemaFields(df.Schema())
	colNames := make([]string, len(fields))
	for i, f := range fields {
		colNames[i] = f.Name
	}
	tableNames := []string{"data"}
	cleanName := cleanFileName(fileName)
	if cleanName != "data" && cleanName != "" {
		tableNames = append(tableNames, cleanName)
	}
	return BuildSQLCorpus(colNames, tableNames)
}

func (m *SQLModel) Focus() {
	m.input.Focus()
}

func (m *SQLModel) SetDataFrame(df *golars.DataFrame, fileName string) {
	m.df = df
	m.sqlCtx = golars.NewSQLContext()
	m.sqlCtx.Register("data", df)

	cleanName := cleanFileName(fileName)
	if cleanName != "data" && cleanName != "" {
		m.sqlCtx.Register(cleanName, df)
	}

	m.result = nil
	m.err = nil
	m.ac.SetCorpus(buildSQLCorpus(df, fileName))
}

func (m *SQLModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 20
	m.table.SetSize(w, max(1, h-8))
}

func (m SQLModel) Update(msg tea.Msg) (SQLModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Autocomplete interaction when visible and input focused
		if m.input.Focused() && m.ac.Visible() {
			switch msg.String() {
			case "tab":
				newText, newCursor := m.ac.Accept(m.input.Value(), m.input.Position())
				m.input.SetValue(newText)
				m.input.SetCursor(newCursor)
				return m, nil
			case "enter":
				newText, newCursor := m.ac.Accept(m.input.Value(), m.input.Position())
				m.input.SetValue(newText)
				m.input.SetCursor(newCursor)
				return m, nil
			case "ctrl+n":
				m.ac.Next()
				return m, nil
			case "ctrl+p":
				m.ac.Prev()
				return m, nil
			case "esc":
				m.ac.Dismiss()
				return m, nil
			}
		}

		switch msg.String() {
		case "enter":
			if m.input.Focused() {
				m.ac.Dismiss()
				return m.executeQuery()
			}
		case "esc":
			if m.input.Focused() {
				m.ac.Dismiss()
				m.input.Blur()
				return m, nil
			}
			m.input.Focus()
			return m, textinput.Blink
		case "ctrl+l":
			m.result = nil
			m.err = nil
			m.ac.Dismiss()
			m.input.Focus()
			return m, textinput.Blink
		case "up":
			if m.input.Focused() {
				if m.ac.Visible() {
					m.ac.Prev()
					return m, nil
				}
				if len(m.history) > 0 {
					if m.histIdx < len(m.history)-1 {
						m.histIdx++
					}
					m.input.SetValue(m.history[len(m.history)-1-m.histIdx])
				}
				return m, nil
			}
		case "down":
			if m.input.Focused() {
				if m.ac.Visible() {
					m.ac.Next()
					return m, nil
				}
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

		// When input is not focused and we have results, delegate to table
		if !m.input.Focused() && m.result != nil {
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	}

	prevValue := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Only re-trigger autocomplete when the text actually changed
	if m.input.Focused() && m.input.Value() != prevValue {
		m.ac.ClearJustAccepted()
		m.ac.Update(m.input.Value(), m.input.Position())
	}

	return m, cmd
}

func (m SQLModel) executeQuery() (SQLModel, tea.Cmd) {
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		return m, nil
	}

	result, err := m.sqlCtx.Execute(query)
	if err != nil {
		m.err = err
		m.result = nil
	} else {
		m.err = nil
		m.result = result
		m.table = NewTableModel(result)
		m.table.SetSize(m.width, max(1, m.height-8))
		m.history = append(m.history, query)
		m.histIdx = -1
		m.input.Blur()
	}
	return m, nil
}

func (m SQLModel) View() string {
	var b strings.Builder

	b.WriteString(statHeaderStyle.Render("SQL Query") + "\n\n")
	b.WriteString(promptStyle.Render("  SQL> ") + m.input.View() + "\n")

	// Show autocomplete dropdown
	if m.input.Focused() && m.ac.Visible() {
		b.WriteString(m.ac.View() + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n")
	}

	if m.result != nil {
		h, w := m.result.Shape()
		b.WriteString(successStyle.Render(fmt.Sprintf("  Result: %d rows × %d columns", h, w)))
		if m.input.Focused() {
			b.WriteString(helpStyle.Render("  (esc to browse results)"))
		} else {
			b.WriteString(helpStyle.Render("  (esc to edit query)"))
		}
		b.WriteString("\n")
		b.WriteString(m.table.View())
	} else if m.err == nil {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Table name: \"data\" │ enter:execute  ctrl+l:clear  ↑↓:history  tab:complete") + "\n")
		b.WriteString(helpStyle.Render("  Example queries:") + "\n")
		b.WriteString(helpStyle.Render("    SELECT * FROM data LIMIT 10") + "\n")
		b.WriteString(helpStyle.Render("    SELECT col1, AVG(col2) FROM data GROUP BY col1") + "\n")
		b.WriteString(helpStyle.Render("    SELECT * FROM data WHERE col1 > 100 ORDER BY col2 DESC") + "\n\n")

		b.WriteString(statHeaderStyle.Render("Schema Reference") + "\n")
		fields := schemaFields(m.df.Schema())
		for _, f := range fields {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %-20s %s", f.Name, shortTypeName(f.Dtype))) + "\n")
		}
	}

	return lipgloss.NewStyle().Width(m.width).MaxHeight(m.height).Render(b.String())
}
