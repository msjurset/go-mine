package app

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type TableModel struct {
	df        *golars.DataFrame
	page      int
	pageSize  int
	cursorRow int
	colOffset int
	sortCol   int // -1 = none
	sortDesc  bool
	sortedDF  *golars.DataFrame
	width     int
	height    int

	// Row detail overlay
	showDetail    bool
	detailScrollY int
}

func NewTableModel(df *golars.DataFrame) TableModel {
	return TableModel{
		df:       df,
		sortedDF: df,
		sortCol:  -1,
		pageSize: 20,
	}
}

func (m *TableModel) SetDataFrame(df *golars.DataFrame) {
	m.df = df
	m.sortedDF = df
	m.page = 0
	m.cursorRow = 0
	m.sortCol = -1
	m.showDetail = false
}

func (m *TableModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.pageSize = max(1, h-6) // header + type row + borders + help
}

func (m TableModel) Update(msg tea.Msg) (TableModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Row detail overlay handles its own keys
		if m.showDetail {
			switch msg.String() {
			case "esc", "enter", "q":
				m.showDetail = false
				m.detailScrollY = 0
			case "j", "down":
				m.detailScrollY++
			case "k", "up":
				if m.detailScrollY > 0 {
					m.detailScrollY--
				}
			case "pgdown":
				m.detailScrollY += 10
			case "pgup":
				m.detailScrollY = max(0, m.detailScrollY-10)
			}
			return m, nil
		}

		totalRows := m.sortedDF.Height()
		totalPages := m.totalPages()

		switch msg.String() {
		case "enter":
			m.showDetail = true
			m.detailScrollY = 0
			return m, nil
		case "j", "down":
			if m.cursorRow < m.pageSize-1 && m.page*m.pageSize+m.cursorRow < totalRows-1 {
				m.cursorRow++
			} else if m.page < totalPages-1 {
				m.page++
				m.cursorRow = 0
			}
		case "k", "up":
			if m.cursorRow > 0 {
				m.cursorRow--
			} else if m.page > 0 {
				m.page--
				m.cursorRow = m.pageSize - 1
			}
		case "l", "right":
			if m.colOffset < m.sortedDF.Width()-1 {
				m.colOffset++
			}
		case "h", "left":
			if m.colOffset > 0 {
				m.colOffset--
			}
		case "pgdown", "ctrl+d":
			if m.page < totalPages-1 {
				m.page++
				m.cursorRow = 0
			}
		case "pgup", "ctrl+u":
			if m.page > 0 {
				m.page--
				m.cursorRow = 0
			}
		case "g":
			m.page = 0
			m.cursorRow = 0
		case "G":
			m.page = totalPages - 1
			m.cursorRow = min(m.pageSize-1, totalRows-1-(totalPages-1)*m.pageSize)
		case "s":
			m.cycleSort()
		case "S":
			m.sortCol = -1
			m.sortedDF = m.df
		}
	}
	return m, nil
}

func (m *TableModel) cycleSort() {
	schema := m.sortedDF.Schema()
	if schema == nil || m.colOffset >= m.df.Width() {
		return
	}

	currentColIdx := m.colOffset

	if m.sortCol == currentColIdx && !m.sortDesc {
		m.sortDesc = true
	} else if m.sortCol == currentColIdx && m.sortDesc {
		m.sortCol = -1
		m.sortedDF = m.df
		m.page = 0
		m.cursorRow = 0
		return
	} else {
		m.sortCol = currentColIdx
		m.sortDesc = false
	}

	colName := m.df.Schema().Field(m.sortCol).Name
	sorted, err := m.df.Sort(colName, m.sortDesc)
	if err == nil {
		m.sortedDF = sorted
		m.page = 0
		m.cursorRow = 0
	}
}

func (m TableModel) View() string {
	if m.sortedDF == nil || m.sortedDF.IsEmpty() {
		return infoStyle.Render("  No data to display")
	}
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	tableContent := m.renderTable()

	if m.showDetail {
		overlay := m.renderRowDetail()
		return m.overlayCenter(tableContent, overlay)
	}

	return tableContent
}

func (m TableModel) renderTable() string {
	schema := m.sortedDF.Schema()
	fields := schemaFields(schema)
	totalCols := len(fields)

	colWidths := m.calcColumnWidths(fields)
	visibleEnd := len(colWidths)
	if visibleEnd == 0 {
		return infoStyle.Render("  Terminal too narrow to display data")
	}

	start := m.page * m.pageSize
	end := min(start+m.pageSize, m.sortedDF.Height())
	pageDF := m.sortedDF.Slice(start, end)

	cursorRow := m.cursorRow
	if cursorRow >= pageDF.Height() {
		cursorRow = pageDF.Height() - 1
	}
	if cursorRow < 0 {
		cursorRow = 0
	}

	var rows []string

	// Header row
	var headerCells []string
	for i, cw := range colWidths {
		idx := m.colOffset + i
		name := fields[idx].Name
		if m.sortCol == idx {
			if m.sortDesc {
				name += " ▼"
			} else {
				name += " ▲"
			}
		}
		if idx == m.colOffset {
			name = "▸ " + name
		}
		headerCells = append(headerCells, headerStyle.Width(cw).MaxWidth(cw).Render(truncate(name, cw-2)))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, headerCells...))

	// Type row
	var typeCells []string
	for i, cw := range colWidths {
		idx := m.colOffset + i
		typeName := shortTypeName(fields[idx].Dtype)
		typeCells = append(typeCells, typeRowStyle.Width(cw).MaxWidth(cw).Render(typeName))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, typeCells...))

	// Separator
	sepParts := make([]string, len(colWidths))
	for i, cw := range colWidths {
		sepParts[i] = lipgloss.NewStyle().Width(cw).Render(strings.Repeat("─", cw))
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, sepParts...))

	// Data rows
	for rowIdx := 0; rowIdx < pageDF.Height(); rowIdx++ {
		var cells []string
		for i, cw := range colWidths {
			colIdx := m.colOffset + i
			col := pageDF.ColumnByIndex(colIdx)
			val := formatCellValue(col, rowIdx)

			style := cellStyle.Width(cw).MaxWidth(cw)
			if rowIdx == cursorRow {
				style = selectedRowStyle.Width(cw).MaxWidth(cw)
			} else if col.IsNull(rowIdx) {
				style = nullStyle.Width(cw).MaxWidth(cw)
			}
			cells = append(cells, style.Render(truncate(val, cw-2)))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, cells...))
	}

	table := lipgloss.JoinVertical(lipgloss.Left, rows...)

	totalPages := m.totalPages()
	absRow := start + cursorRow + 1
	footer := helpStyle.Render(fmt.Sprintf(
		"  Page %d/%d │ Row %d/%d │ Col %d-%d/%d │ ↑↓:nav ←→:scroll s:sort enter:detail g/G:top/bottom",
		m.page+1, totalPages, absRow, m.sortedDF.Height(),
		m.colOffset+1, min(m.colOffset+len(colWidths), totalCols), totalCols,
	))

	return table + "\n" + footer
}

func (m TableModel) renderRowDetail() string {
	absRowIdx := m.page*m.pageSize + m.cursorRow
	if absRowIdx >= m.sortedDF.Height() {
		return ""
	}

	fields := schemaFields(m.sortedDF.Schema())

	// Find the max field name length for alignment
	maxName := 0
	for _, f := range fields {
		if len(f.Name) > maxName {
			maxName = len(f.Name)
		}
	}

	// Size the box — use up to 80% of terminal, min 50
	boxWidth := min(m.width*4/5, m.width-8)
	if boxWidth < 50 {
		boxWidth = 50
	}

	// Available width for the value column (inside box: border + padding eats ~6 chars)
	prefixWidth := maxName + 1 + 2 + 7 // "  name  " + space + "type  " + space
	innerWidth := boxWidth - 6          // subtract border (2) + padding (2*2)
	valWidth := innerWidth - prefixWidth
	if valWidth < 10 {
		valWidth = 10
	}

	// Build the detail lines, wrapping long values
	title := fmt.Sprintf(" Row %d of %d ", absRowIdx+1, m.sortedDF.Height())
	indent := strings.Repeat(" ", prefixWidth)
	var lines []string
	for _, f := range fields {
		col := getColumn(m.sortedDF, f.Name)
		val := formatCellValue(col, absRowIdx)
		typStr := shortTypeName(f.Dtype)

		label := statusKeyStyle.Render(fmt.Sprintf("  %-*s", maxName+1, f.Name))
		typeTag := typeRowStyle.Render(fmt.Sprintf("%-6s", typStr))

		valStyle := statValueStyle
		if col.IsNull(absRowIdx) {
			valStyle = nullStyle
		}

		// Wrap value into chunks that fit the available width
		wrapped := wrapText(val, valWidth)
		firstLine := label + " " + typeTag + " " + valStyle.Render(wrapped[0])
		lines = append(lines, firstLine)
		for _, cont := range wrapped[1:] {
			lines = append(lines, indent+valStyle.Render(cont))
		}
	}

	// Apply scroll within the detail box
	visibleLines := m.height - 8 // border + title + footer + padding
	if visibleLines < 3 {
		visibleLines = 3
	}

	scrollY := m.detailScrollY
	maxScroll := len(lines) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	endLine := min(scrollY+visibleLines, len(lines))
	visibleContent := strings.Join(lines[scrollY:endLine], "\n")

	// Scroll indicator
	scrollInfo := ""
	if len(lines) > visibleLines {
		scrollInfo = helpStyle.Render(fmt.Sprintf("  showing %d-%d of %d fields", scrollY+1, endLine, len(lines)))
	}

	footer := helpStyle.Render("  esc/enter:close  ↑↓:scroll")
	if scrollInfo != "" {
		footer = scrollInfo + "  │" + footer
	}

	body := visibleContent + "\n\n" + footer

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(boxWidth).
		Render(body)

	// Title on the border
	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorPrimary).
		Padding(0, 1).
		Render(title)

	// Place title over the top border
	boxLines := strings.Split(box, "\n")
	if len(boxLines) > 0 {
		borderWidth := lipgloss.Width(boxLines[0])
		titleWidth := lipgloss.Width(titleRendered)
		if titleWidth+4 < borderWidth {
			borderStyle := lipgloss.NewStyle().Foreground(colorPrimary)
			leadPad := 2 // chars before title
			trailLen := borderWidth - leadPad - titleWidth - 2 // -2 for ╭ and ╮
			if trailLen < 0 {
				trailLen = 0
			}
			boxLines[0] = borderStyle.Render("╭"+strings.Repeat("─", leadPad)) +
				titleRendered +
				borderStyle.Render(strings.Repeat("─", trailLen)+"╮")
		}
		box = strings.Join(boxLines, "\n")
	}

	return box
}

// overlayCenter places the overlay box centered on top of the background content.
func (m TableModel) overlayCenter(bg, overlay string) string {
	bgLines := strings.Split(bg, "\n")
	ovLines := strings.Split(overlay, "\n")

	// Vertical centering
	startRow := (len(bgLines) - len(ovLines)) / 2
	if startRow < 0 {
		startRow = 0
	}

	// Horizontal centering
	ovWidth := 0
	for _, l := range ovLines {
		if w := lipgloss.Width(l); w > ovWidth {
			ovWidth = w
		}
	}
	startCol := (m.width - ovWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Composite: dim the background, overlay the box
	result := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		ovIdx := i - startRow
		if ovIdx >= 0 && ovIdx < len(ovLines) {
			ovLine := ovLines[ovIdx]
			ovW := lipgloss.Width(ovLine)
			// Build: left padding + overlay + rest of bg
			left := padOrTruncate(bgLine, startCol)
			// Dim the left portion
			left = lipgloss.NewStyle().Faint(true).Render(stripAnsi(left))
			right := ""
			rightStart := startCol + ovW
			if rightStart < lipgloss.Width(bgLine) {
				right = lipgloss.NewStyle().Faint(true).Render(
					stripAnsi(sliceVisual(bgLine, rightStart)))
			}
			result[i] = left + ovLine + right
		} else {
			// Dim background rows
			result[i] = lipgloss.NewStyle().Faint(true).Render(stripAnsi(bgLine))
		}
	}

	return strings.Join(result, "\n")
}

// padOrTruncate returns a string that is exactly `width` visible characters,
// padding with spaces or truncating as needed.
func padOrTruncate(s string, width int) string {
	plain := stripAnsi(s)
	if len(plain) >= width {
		return plain[:width]
	}
	return plain + strings.Repeat(" ", width-len(plain))
}

// sliceVisual returns the substring starting at the given visible column.
func sliceVisual(s string, startCol int) string {
	plain := stripAnsi(s)
	if startCol >= len(plain) {
		return ""
	}
	return plain[startCol:]
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (m TableModel) calcColumnWidths(fields []golars.Field) []int {
	available := m.width - 2
	var widths []int
	used := 0

	start := m.page * m.pageSize
	end := min(start+m.pageSize, m.sortedDF.Height())
	pageDF := m.sortedDF.Slice(start, end)

	for i := m.colOffset; i < len(fields) && used < available; i++ {
		col := getColumn(pageDF, fields[i].Name)
		w := calcColWidth(col, fields[i].Name)
		w = min(w, 40)
		w = max(w, 6)
		if used+w > available {
			break
		}
		widths = append(widths, w)
		used += w
	}
	return widths
}

func calcColWidth(s *golars.Series, name string) int {
	w := len(name) + 4
	for i := 0; i < s.Len(); i++ {
		val := formatCellValue(s, i)
		if len(val)+2 > w {
			w = len(val) + 2
		}
	}
	return w
}

func (m TableModel) totalPages() int {
	if m.sortedDF.Height() == 0 {
		return 1
	}
	return int(math.Ceil(float64(m.sortedDF.Height()) / float64(m.pageSize)))
}

func formatCellValue(s *golars.Series, i int) string {
	if s.IsNull(i) {
		return "null"
	}
	dt := s.DataType()
	switch dt {
	case golars.Int8, golars.Int16, golars.Int32, golars.Int64:
		v, _ := s.GetInt64(i)
		return fmt.Sprintf("%d", v)
	case golars.UInt8, golars.UInt16, golars.UInt32, golars.UInt64:
		v, _ := s.GetInt64(i)
		return fmt.Sprintf("%d", v)
	case golars.Float32, golars.Float64:
		v, _ := s.GetFloat64(i)
		if v == math.Trunc(v) && math.Abs(v) < 1e15 {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%.4g", v)
	case golars.Boolean:
		v, _ := s.GetBool(i)
		if v {
			return "true"
		}
		return "false"
	case golars.String:
		v, _ := s.GetString(i)
		return v
	default:
		v, _ := s.GetString(i)
		return v
	}
}

func shortTypeName(dt golars.DataType) string {
	switch dt {
	case golars.Int8:
		return "i8"
	case golars.Int16:
		return "i16"
	case golars.Int32:
		return "i32"
	case golars.Int64:
		return "i64"
	case golars.UInt8:
		return "u8"
	case golars.UInt16:
		return "u16"
	case golars.UInt32:
		return "u32"
	case golars.UInt64:
		return "u64"
	case golars.Float32:
		return "f32"
	case golars.Float64:
		return "f64"
	case golars.Boolean:
		return "bool"
	case golars.String:
		return "str"
	case golars.Date:
		return "date"
	case golars.DateTime:
		return "datetime"
	case golars.Time:
		return "time"
	case golars.Duration:
		return "dur"
	default:
		return "?"
	}
}

// wrapText splits a string into lines of at most maxWidth characters.
// It tries to break at spaces when possible.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 1
	}
	if len(s) <= maxWidth {
		return []string{s}
	}

	var lines []string
	for len(s) > 0 {
		if len(s) <= maxWidth {
			lines = append(lines, s)
			break
		}
		// Try to break at a space
		cut := maxWidth
		if idx := strings.LastIndex(s[:cut], " "); idx > 0 {
			cut = idx
		}
		lines = append(lines, s[:cut])
		s = strings.TrimLeft(s[cut:], " ")
	}
	return lines
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}
