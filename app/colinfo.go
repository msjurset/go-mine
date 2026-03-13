package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type ColInfoModel struct {
	df      *golars.DataFrame
	scrollY int
	width   int
	height  int
}

func NewColInfoModel(df *golars.DataFrame) ColInfoModel {
	return ColInfoModel{df: df}
}

func (m *ColInfoModel) SetDataFrame(df *golars.DataFrame) {
	m.df = df
	m.scrollY = 0
}

func (m *ColInfoModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ColInfoModel) Update(msg tea.Msg) (ColInfoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scrollY++
		case "k", "up":
			if m.scrollY > 0 {
				m.scrollY--
			}
		case "pgdown":
			m.scrollY += 10
		case "pgup":
			m.scrollY = max(0, m.scrollY-10)
		case "g":
			m.scrollY = 0
		}
	}
	return m, nil
}

func (m ColInfoModel) View() string {
	if m.df == nil || m.df.IsEmpty() {
		return infoStyle.Render("  No data to display")
	}
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	fields := schemaFields(m.df.Schema())
	h, _ := m.df.Shape()

	var sections []string

	for i, f := range fields {
		col := getColumn(m.df, f.Name)
		section := m.renderColumnDetail(col, f, i, h)
		sections = append(sections, section)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Apply scrolling
	lines := strings.Split(content, "\n")
	visibleHeight := max(1, m.height-3)
	start, end := clampedScroll(m.scrollY, len(lines), visibleHeight)
	if start < len(lines) {
		content = strings.Join(lines[start:end], "\n")
	}

	footer := helpStyle.Render(fmt.Sprintf(
		"  %d columns │ %d rows │ ↑↓:scroll pgup/pgdn:page g:top",
		len(fields), h,
	))

	return content + "\n" + footer
}

func (m ColInfoModel) renderColumnDetail(s *golars.Series, f golars.Field, idx, totalRows int) string {
	var b strings.Builder

	title := fmt.Sprintf("  [%d] %s", idx+1, f.Name)
	b.WriteString(statHeaderStyle.Render(title) + "\n")

	b.WriteString(fmt.Sprintf("  %-16s %s\n", "Type:", shortTypeName(f.Dtype)))
	b.WriteString(fmt.Sprintf("  %-16s %d\n", "Count:", s.Count()))
	b.WriteString(fmt.Sprintf("  %-16s %d\n", "Null count:", s.NullCount()))

	nullPct := 0.0
	if totalRows > 0 {
		nullPct = float64(s.NullCount()) / float64(totalRows) * 100
	}
	b.WriteString(fmt.Sprintf("  %-16s %.1f%%\n", "Null %:", nullPct))
	b.WriteString(fmt.Sprintf("  %-16s %d\n", "Unique:", s.NUnique()))

	if isNumeric(f.Dtype) {
		if mean, ok := s.Mean(); ok {
			b.WriteString(fmt.Sprintf("  %-16s %.6g\n", "Mean:", mean))
		}
		if std, ok := s.Std(); ok {
			b.WriteString(fmt.Sprintf("  %-16s %.6g\n", "Std:", std))
		}
		if minV, ok := s.Min(); ok {
			b.WriteString(fmt.Sprintf("  %-16s %.6g\n", "Min:", minV))
		}
		if maxV, ok := s.Max(); ok {
			b.WriteString(fmt.Sprintf("  %-16s %.6g\n", "Max:", maxV))
		}
		if sum, ok := s.Sum(); ok {
			b.WriteString(fmt.Sprintf("  %-16s %.6g\n", "Sum:", sum))
		}

		b.WriteString(m.renderHistogram(s))
	}

	if f.Dtype == golars.String {
		b.WriteString("  Sample values:  ")
		n := min(5, s.Len())
		var samples []string
		for i := 0; i < n; i++ {
			if !s.IsNull(i) {
				v, _ := s.GetString(i)
				samples = append(samples, fmt.Sprintf("%q", truncate(v, 30)))
			}
		}
		b.WriteString(strings.Join(samples, ", ") + "\n")
	}

	b.WriteString("  " + strings.Repeat("─", 40) + "\n")
	return b.String()
}

func (m ColInfoModel) renderHistogram(s *golars.Series) string {
	minV, minOk := s.Min()
	maxV, maxOk := s.Max()
	if !minOk || !maxOk || minV == maxV {
		return ""
	}

	bins := 20
	counts := make([]int, bins)
	binWidth := (maxV - minV) / float64(bins)

	for i := 0; i < s.Len(); i++ {
		if s.IsNull(i) {
			continue
		}
		var v float64
		switch s.DataType() {
		case golars.Float32, golars.Float64:
			v, _ = s.GetFloat64(i)
		default:
			iv, ok := s.GetInt64(i)
			if !ok {
				continue
			}
			v = float64(iv)
		}
		bin := int((v - minV) / binWidth)
		if bin < 0 {
			bin = 0
		}
		if bin >= bins {
			bin = bins - 1
		}
		counts[bin]++
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	if maxCount == 0 {
		return ""
	}

	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var spark strings.Builder
	spark.WriteString("  Distribution:   ")
	for _, c := range counts {
		idx := int(float64(c) / float64(maxCount) * float64(len(bars)-1))
		spark.WriteRune(bars[idx])
	}
	spark.WriteString("\n")
	return spark.String()
}
