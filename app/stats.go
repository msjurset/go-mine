package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type StatsModel struct {
	df       *golars.DataFrame
	scrollY  int
	width    int
	height   int
	colIndex int // selected column for detail
}

func NewStatsModel(df *golars.DataFrame) StatsModel {
	return StatsModel{df: df}
}

func (m *StatsModel) SetDataFrame(df *golars.DataFrame) {
	m.df = df
	m.scrollY = 0
	m.colIndex = 0
}

func (m *StatsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m StatsModel) Update(msg tea.Msg) (StatsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.colIndex < m.df.Width()-1 {
				m.colIndex++
			}
			m.ensureVisible()
		case "k", "up":
			if m.colIndex > 0 {
				m.colIndex--
			}
			m.ensureVisible()
		case "pgdown":
			m.colIndex = min(m.colIndex+5, m.df.Width()-1)
			m.ensureVisible()
		case "pgup":
			m.colIndex = max(m.colIndex-5, 0)
			m.ensureVisible()
		}
	}
	return m, nil
}

func (m *StatsModel) ensureVisible() {
	cardHeight := 8
	cardTop := m.colIndex * cardHeight
	visibleHeight := m.height - 4
	if cardTop < m.scrollY {
		m.scrollY = cardTop
	} else if cardTop+cardHeight > m.scrollY+visibleHeight {
		m.scrollY = cardTop + cardHeight - visibleHeight
	}
}

func (m StatsModel) View() string {
	if m.df == nil || m.df.IsEmpty() {
		return infoStyle.Render("  No data to display")
	}
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	fields := schemaFields(m.df.Schema())
	h, w := m.df.Shape()

	// Dataset overview
	overview := statHeaderStyle.Render("Dataset Overview") + "\n"
	overview += statLabelStyle.Render("Rows:") + statValueStyle.Render(fmt.Sprintf(" %d", h)) + "\n"
	overview += statLabelStyle.Render("Columns:") + statValueStyle.Render(fmt.Sprintf(" %d", w)) + "\n"

	// Describe output
	descDF := m.df.Describe()
	if descDF != nil {
		overview += "\n" + statHeaderStyle.Render("Describe (numeric columns)") + "\n"
		overview += renderMiniTable(descDF, m.width/2-4)
	}

	// Per-column detail cards
	var cards []string
	for i, f := range fields {
		col := getColumn(m.df, f.Name)
		card := m.renderColumnCard(col, f, i == m.colIndex)
		cards = append(cards, card)
	}

	rightPanel := lipgloss.JoinVertical(lipgloss.Left, cards...)

	// Apply scroll
	rightLines := strings.Split(rightPanel, "\n")
	visibleHeight := max(1, m.height-4)
	scrollY := m.scrollY
	maxScroll := len(rightLines) - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	endLine := min(scrollY+visibleHeight, len(rightLines))
	if scrollY < len(rightLines) && scrollY <= endLine {
		rightPanel = strings.Join(rightLines[scrollY:endLine], "\n")
	}

	leftPanel := lipgloss.NewStyle().
		Width(m.width/2 - 2).
		Render(overview)

	rightPanelStyled := lipgloss.NewStyle().
		Width(m.width/2 - 2).
		Render(rightPanel)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " │ ", rightPanelStyled)

	footer := helpStyle.Render(fmt.Sprintf(
		"  Column %d/%d │ ↑↓:select column pgup/pgdn:page",
		m.colIndex+1, len(fields),
	))

	return content + "\n" + footer
}

func (m StatsModel) renderColumnCard(s *golars.Series, f golars.Field, selected bool) string {
	var b strings.Builder

	name := f.Name
	if selected {
		name = "▸ " + name
	}
	b.WriteString(statHeaderStyle.Render(name) + "\n")
	b.WriteString(statLabelStyle.Render("Type:") + statValueStyle.Render(fmt.Sprintf(" %s", shortTypeName(f.Dtype))) + "\n")
	b.WriteString(statLabelStyle.Render("Count:") + statValueStyle.Render(fmt.Sprintf(" %d", s.Count())) + "\n")
	b.WriteString(statLabelStyle.Render("Nulls:") + statValueStyle.Render(fmt.Sprintf(" %d", s.NullCount())) + "\n")
	b.WriteString(statLabelStyle.Render("Unique:") + statValueStyle.Render(fmt.Sprintf(" %d", s.NUnique())) + "\n")

	if isNumeric(f.Dtype) {
		if mean, ok := s.Mean(); ok {
			b.WriteString(statLabelStyle.Render("Mean:") + statValueStyle.Render(fmt.Sprintf(" %.4g", mean)) + "\n")
		}
		if std, ok := s.Std(); ok {
			b.WriteString(statLabelStyle.Render("Std:") + statValueStyle.Render(fmt.Sprintf(" %.4g", std)) + "\n")
		}
		if minV, ok := s.Min(); ok {
			b.WriteString(statLabelStyle.Render("Min:") + statValueStyle.Render(fmt.Sprintf(" %.4g", minV)) + "\n")
		}
		if maxV, ok := s.Max(); ok {
			b.WriteString(statLabelStyle.Render("Max:") + statValueStyle.Render(fmt.Sprintf(" %.4g", maxV)) + "\n")
		}
		if sum, ok := s.Sum(); ok {
			b.WriteString(statLabelStyle.Render("Sum:") + statValueStyle.Render(fmt.Sprintf(" %.4g", sum)) + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func renderMiniTable(df *golars.DataFrame, maxWidth int) string {
	if df == nil || df.IsEmpty() {
		return ""
	}

	fields := schemaFields(df.Schema())
	colWidths := make([]int, len(fields))
	for i, f := range fields {
		colWidths[i] = len(f.Name) + 2
		col := df.ColumnByIndex(i)
		for j := 0; j < col.Len(); j++ {
			val := formatCellValue(col, j)
			if len(val)+2 > colWidths[i] {
				colWidths[i] = len(val) + 2
			}
		}
		colWidths[i] = min(colWidths[i], 14)
	}

	var rows []string

	// Header
	var hdr []string
	for i, f := range fields {
		hdr = append(hdr, headerStyle.Width(colWidths[i]).MaxWidth(colWidths[i]).Render(truncate(f.Name, colWidths[i]-2)))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, hdr...))

	// Data
	for r := 0; r < df.Height(); r++ {
		var cells []string
		for i := range fields {
			col := df.ColumnByIndex(i)
			val := formatCellValue(col, r)
			cells = append(cells, cellStyle.Width(colWidths[i]).MaxWidth(colWidths[i]).Render(truncate(val, colWidths[i]-2)))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func isNumeric(dt golars.DataType) bool {
	switch dt {
	case golars.Int8, golars.Int16, golars.Int32, golars.Int64,
		golars.UInt8, golars.UInt16, golars.UInt32, golars.UInt64,
		golars.Float32, golars.Float64:
		return true
	}
	return false
}
