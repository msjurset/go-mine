package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type ExportDoneMsg struct{ Path string }
type ExportErrorMsg struct{ Err error }

type ExportModel struct {
	df     *golars.DataFrame
	input  textinput.Model
	active bool
	err    error
	done   string
	width  int
}

func NewExportModel() ExportModel {
	ti := textinput.New()
	ti.Placeholder = "output.csv (supports .csv, .parquet, .json)"
	ti.CharLimit = 256
	ti.Width = 60
	return ExportModel{input: ti}
}

func (m *ExportModel) SetDataFrame(df *golars.DataFrame) {
	m.df = df
}

func (m *ExportModel) Open() tea.Cmd {
	m.active = true
	m.err = nil
	m.done = ""
	m.input.SetValue("")
	m.input.Focus()
	return textinput.Blink
}

func (m *ExportModel) Close() {
	m.active = false
	m.input.Blur()
}

func (m ExportModel) Active() bool {
	return m.active
}

func (m ExportModel) Update(msg tea.Msg) (ExportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m.doExport()
		case "esc":
			m.Close()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ExportModel) doExport() (ExportModel, tea.Cmd) {
	path := strings.TrimSpace(m.input.Value())
	if path == "" {
		m.err = fmt.Errorf("no file path provided")
		return m, nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	var err error

	switch ext {
	case ".csv":
		err = golars.WriteCSVFile(m.df, path)
	case ".parquet":
		err = golars.WriteParquetFile(m.df, path)
	case ".json":
		err = golars.WriteJSONFile(m.df, path)
	default:
		m.err = fmt.Errorf("unsupported format: %s (use .csv, .parquet, .json)", ext)
		return m, nil
	}

	if err != nil {
		m.err = fmt.Errorf("export failed: %w", err)
		return m, nil
	}

	m.done = path
	m.err = nil
	return m, nil
}

func (m ExportModel) View() string {
	var b strings.Builder

	b.WriteString(statHeaderStyle.Render("Export Data") + "\n\n")

	h, w := m.df.Shape()
	b.WriteString(helpStyle.Render(fmt.Sprintf("  Exporting %d rows x %d columns", h, w)) + "\n\n")

	b.WriteString(promptStyle.Render("  Path: ") + m.input.View() + "\n\n")

	b.WriteString(helpStyle.Render("  Supported formats: .csv, .parquet, .json") + "\n")
	b.WriteString(helpStyle.Render("  enter:export  esc:cancel") + "\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n")
	}
	if m.done != "" {
		b.WriteString(successStyle.Render(fmt.Sprintf("  Exported to %s", m.done)) + "\n")
	}

	return lipgloss.NewStyle().Width(m.width).Render(b.String())
}
