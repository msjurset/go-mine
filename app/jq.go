package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/itchyny/gojq"
	"github.com/msjurset/golars"
)

const maxJQResults = 10000

type JQModel struct {
	df        *golars.DataFrame
	rawJSON   []interface{} // raw nested JSON (when loaded from .json file)
	jsonCache []interface{} // lazy conversion from df
	input     textinput.Model
	history   []string
	histIdx   int
	result    *golars.DataFrame // if output is tabular
	table     TableModel
	treeView  JSONTreeView // syntax-highlighted JSON tree for results
	showTree  bool         // true = tree view, false = table view (when tabular)
	err       error
	width     int
	height    int
	ac        Autocomplete
}

func NewJQModel(df *golars.DataFrame) JQModel {
	ti := textinput.New()
	ti.Placeholder = `.[] | select(.age > 30) | {name, age}`
	ti.CharLimit = 512
	ti.Width = 80

	ac := NewAutocomplete()
	ac.SetCorpus(buildJQCorpus(df))

	return JQModel{
		df:       df,
		histIdx:  -1,
		input:    ti,
		treeView: NewJSONTreeView(),
		showTree: true,
		ac:       ac,
	}
}

func buildJQCorpus(df *golars.DataFrame) []Suggestion {
	fields := schemaFields(df.Schema())
	colNames := make([]string, len(fields))
	for i, f := range fields {
		colNames[i] = f.Name
	}
	return BuildJQCorpus(colNames)
}

func (m *JQModel) Focus() {
	m.input.Focus()
}

func (m *JQModel) SetRawJSON(raw []interface{}) {
	m.rawJSON = raw
	m.jsonCache = nil
}

func (m *JQModel) SetDataFrame(df *golars.DataFrame) {
	m.df = df
	m.jsonCache = nil
	m.result = nil
	m.treeView = NewJSONTreeView()
	m.showTree = true
	m.err = nil
	m.ac.SetCorpus(buildJQCorpus(df))
}

func (m *JQModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 20
	m.table.SetSize(w, max(1, h-8))
	m.treeView.SetSize(w, max(1, h-8))
}

func (m JQModel) Update(msg tea.Msg) (JQModel, tea.Cmd) {
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
			// Toggle expand/collapse in tree view
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.Toggle()
				return m, nil
			}
		case " ":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.Toggle()
				return m, nil
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
			m.treeView = NewJSONTreeView()
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
		case "j":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.CursorDown()
				return m, nil
			}
		case "k":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.CursorUp()
				return m, nil
			}
		case "E":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.ExpandAll()
				return m, nil
			}
		case "C":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.CollapseAll()
				return m, nil
			}
		case "pgdown", "ctrl+f":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.PageDown()
				return m, nil
			}
		case "pgup", "ctrl+b":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.PageUp()
				return m, nil
			}
		case "g":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.GoToTop()
				return m, nil
			}
		case "G":
			if !m.input.Focused() && m.treeView.HasData() {
				m.treeView.GoToBottom()
				return m, nil
			}
		case "t":
			// Toggle between tree and table view (only when tabular results available)
			if !m.input.Focused() && m.result != nil && m.treeView.HasData() {
				m.showTree = !m.showTree
				return m, nil
			}
		}

		// Delegate to table when in table mode
		if !m.input.Focused() && !m.showTree && m.result != nil {
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

func (m JQModel) executeQuery() (JQModel, tea.Cmd) {
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		return m, nil
	}

	// Lazy-build JSON cache: prefer raw JSON (preserves nesting) over DataFrame conversion
	if m.jsonCache == nil {
		if m.rawJSON != nil {
			m.jsonCache = m.rawJSON
		} else {
			m.jsonCache = dataFrameToJSON(m.df)
		}
	}

	parsed, err := gojq.Parse(query)
	if err != nil {
		m.err = fmt.Errorf("parse: %w", err)
		m.result = nil
		m.treeView = NewJSONTreeView()
		return m, nil
	}

	code, err := gojq.Compile(parsed)
	if err != nil {
		m.err = fmt.Errorf("compile: %w", err)
		m.result = nil
		m.treeView = NewJSONTreeView()
		return m, nil
	}

	var results []interface{}
	truncated := false
	iter := code.Run(m.jsonCache)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			m.err = fmt.Errorf("run: %w", err)
			m.result = nil
			m.treeView = NewJSONTreeView()
			return m, nil
		}
		results = append(results, v)
		if len(results) >= maxJQResults {
			truncated = true
			break
		}
	}

	if len(results) == 0 {
		m.err = fmt.Errorf("query returned no results")
		m.result = nil
		m.treeView = NewJSONTreeView()
	} else {
		m.err = nil
		// Always build tree view for all results
		m.treeView = NewJSONTreeView()
		m.treeView.SetSize(m.width, max(1, m.height-8))
		m.treeView.SetData(results, truncated)
		m.showTree = true

		// Also try tabular conversion so user can toggle to table view
		df, dfErr := jsonToDataFrame(results)
		if dfErr == nil {
			m.result = df
			m.table = NewTableModel(df)
			m.table.SetSize(m.width, max(1, m.height-8))
		} else {
			m.result = nil
		}
	}

	m.history = append(m.history, query)
	m.histIdx = -1
	m.input.Blur()
	return m, nil
}

func (m JQModel) View() string {
	var b strings.Builder

	b.WriteString(statHeaderStyle.Render("JQ Query") + "\n\n")
	b.WriteString(promptStyle.Render("  jq> ") + m.input.View() + "\n")

	// Show autocomplete dropdown
	if m.input.Focused() && m.ac.Visible() {
		b.WriteString(m.ac.View() + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n")
	}

	if m.treeView.HasData() {
		// Show result info and view toggle hint
		if m.result != nil {
			h, w := m.result.Shape()
			b.WriteString(successStyle.Render(fmt.Sprintf("  Result: %d rows × %d columns", h, w)))
		}
		if m.input.Focused() {
			b.WriteString(helpStyle.Render("  (esc to browse results)"))
		} else {
			b.WriteString(helpStyle.Render("  (esc to edit query)"))
			if m.result != nil {
				if m.showTree {
					b.WriteString(helpStyle.Render("  (t: switch to table)"))
				} else {
					b.WriteString(helpStyle.Render("  (t: switch to tree)"))
				}
			}
		}
		b.WriteString("\n")

		if !m.showTree && m.result != nil {
			b.WriteString(m.table.View())
		} else {
			b.WriteString(m.treeView.View())
		}
	} else if m.err == nil {
		// Show help and schema when no results
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Input is the full dataset as a JSON array │ enter:execute  ctrl+l:clear  ↑↓:history  tab:complete") + "\n")
		b.WriteString(helpStyle.Render("  Example queries:") + "\n")
		b.WriteString(helpStyle.Render(`    .[] | select(.age > 30)`) + "\n")
		b.WriteString(helpStyle.Render(`    [.[] | .name] | unique`) + "\n")
		b.WriteString(helpStyle.Render(`    group_by(.department) | map({key: .[0].department, count: length})`) + "\n")
		b.WriteString(helpStyle.Render(`    .[:5]`) + "\n\n")

		b.WriteString(statHeaderStyle.Render("Schema Reference") + "\n")
		fields := schemaFields(m.df.Schema())
		for _, f := range fields {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %-20s %s", f.Name, shortTypeName(f.Dtype))) + "\n")
		}
	}

	return lipgloss.NewStyle().Width(m.width).MaxHeight(m.height).Render(b.String())
}
