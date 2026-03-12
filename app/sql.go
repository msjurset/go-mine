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
}

func NewSQLModel(df *golars.DataFrame, fileName string) SQLModel {
	ti := textinput.New()
	ti.Placeholder = `SELECT * FROM data WHERE ... GROUP BY ... ORDER BY ... LIMIT 20`
	ti.CharLimit = 512
	ti.Width = 80

	ctx := golars.NewSQLContext()
	ctx.Register("data", df)

	// Also register with a cleaned-up filename
	cleanName := strings.TrimSuffix(fileName, ".csv")
	cleanName = strings.TrimSuffix(cleanName, ".parquet")
	cleanName = strings.TrimSuffix(cleanName, ".json")
	cleanName = strings.TrimSuffix(cleanName, ".tsv")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	if cleanName != "data" && cleanName != "" {
		ctx.Register(cleanName, df)
	}

	return SQLModel{
		df:      df,
		sqlCtx:  ctx,
		input:   ti,
		histIdx: -1,
	}
}

func (m *SQLModel) Focus() {
	m.input.Focus()
}

func (m *SQLModel) SetDataFrame(df *golars.DataFrame, fileName string) {
	m.df = df
	m.sqlCtx = golars.NewSQLContext()
	m.sqlCtx.Register("data", df)

	cleanName := strings.TrimSuffix(fileName, ".csv")
	cleanName = strings.TrimSuffix(cleanName, ".parquet")
	cleanName = strings.TrimSuffix(cleanName, ".json")
	cleanName = strings.TrimSuffix(cleanName, ".tsv")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	if cleanName != "data" && cleanName != "" {
		m.sqlCtx.Register(cleanName, df)
	}

	m.result = nil
	m.err = nil
}

func (m *SQLModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 20
	// Result table gets the space below the input/help area (about 8 lines)
	m.table.SetSize(w, max(1, h-8))
}

func (m SQLModel) Update(msg tea.Msg) (SQLModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Focused() {
				return m.executeQuery()
			}
		case "esc":
			if m.input.Focused() {
				m.input.Blur()
				return m, nil
			}
			// Re-focus input from result table
			m.input.Focus()
			return m, textinput.Blink
		case "ctrl+l":
			m.result = nil
			m.err = nil
			m.input.Focus()
			return m, textinput.Blink
		case "up":
			if m.input.Focused() {
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

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
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
		// Blur input so user can navigate results immediately
		m.input.Blur()
	}
	return m, nil
}

func (m SQLModel) View() string {
	var b strings.Builder

	b.WriteString(statHeaderStyle.Render("SQL Query") + "\n\n")
	b.WriteString(promptStyle.Render("  SQL> ") + m.input.View() + "\n")

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
		// Show help and schema when no results
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Table name: \"data\" │ enter:execute  ctrl+l:clear  ↑↓:history  esc:unfocus") + "\n")
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

	return lipgloss.NewStyle().Width(m.width).Render(b.String())
}
