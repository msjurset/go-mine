package app

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type SearchState int

const (
	SearchInactive  SearchState = iota
	SearchInput                 // typing pattern
	SearchNavigate              // pattern set, n/N navigate
)

// DataMatch represents a match in the DataFrame: which row and column.
type DataMatch struct {
	Row int
	Col int
}

type SearchModel struct {
	input   textinput.Model
	state   SearchState
	pattern string
	regex   *regexp.Regexp
	err     error
	matches []DataMatch // matches found in DataFrame
	current int         // current match index for navigation
	width   int
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256
	ti.Width = 40
	return SearchModel{
		input: ti,
		state: SearchInactive,
	}
}

func (m *SearchModel) Open() tea.Cmd {
	m.state = SearchInput
	m.pattern = ""
	m.regex = nil
	m.err = nil
	m.matches = nil
	m.current = 0
	m.input.SetValue("")
	m.input.Focus()
	return textinput.Blink
}

func (m *SearchModel) Close() {
	m.state = SearchInactive
	m.input.Blur()
	m.pattern = ""
	m.regex = nil
	m.err = nil
	m.matches = nil
	m.current = 0
}

func (m SearchModel) Active() bool {
	return m.state != SearchInactive
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.input.Value()
			if val == "" {
				m.Close()
				return m, nil
			}
			re, err := regexp.Compile("(?i)" + val)
			if err != nil {
				m.err = fmt.Errorf("invalid regex: %v", err)
				return m, nil
			}
			m.pattern = val
			m.regex = re
			m.err = nil
			m.state = SearchNavigate
			m.input.Blur()
			m.current = 0
			return m, nil
		case "esc":
			m.Close()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// ScanDataFrame searches all cell string values in the DataFrame and
// populates m.matches with (row, col) pairs. Call this when entering
// SearchNavigate or when the DataFrame changes.
func (m *SearchModel) ScanDataFrame(df *golars.DataFrame) {
	m.matches = nil
	if m.regex == nil || df == nil {
		return
	}

	height := df.Height()
	width := df.Width()

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			c := df.ColumnByIndex(col)
			val := formatCellValueForSearch(c, row)
			if m.regex.MatchString(val) {
				m.matches = append(m.matches, DataMatch{Row: row, Col: col})
			}
		}
	}

	// Clamp current
	if len(m.matches) > 0 && m.current >= len(m.matches) {
		m.current = 0
	}
}

// formatCellValueForSearch returns the string representation of a cell.
func formatCellValueForSearch(col *golars.Series, row int) string {
	if col.IsNull(row) {
		return "null"
	}
	// Use the same approach as formatCellValue but always wide format
	return formatCellValue(col, row, true)
}

func (m *SearchModel) NextMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.current = (m.current + 1) % len(m.matches)
}

func (m *SearchModel) PrevMatch() {
	if len(m.matches) == 0 {
		return
	}
	m.current = (m.current + len(m.matches) - 1) % len(m.matches)
}

// CurrentMatch returns the current DataMatch, or (-1, -1) if none.
func (m SearchModel) CurrentMatch() (row, col int) {
	if len(m.matches) == 0 {
		return -1, -1
	}
	dm := m.matches[m.current]
	return dm.Row, dm.Col
}

// stripANSI removes all ANSI escape sequences from a string.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// HighlightContent post-processes rendered view content to apply search
// highlights. All matches get amber highlight; matches on activeLine get
// the active (red) highlight. Pass activeLine=-1 to use amber everywhere.
func (m SearchModel) HighlightContent(content string, activeLine int) string {
	if m.regex == nil {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string

	for i, line := range lines {
		plain := stripANSI(line)
		locs := m.regex.FindAllStringIndex(plain, -1)
		if len(locs) == 0 {
			result = append(result, line)
			continue
		}

		isActive := i == activeLine
		var b strings.Builder
		prev := 0
		for _, loc := range locs {
			if prev < loc[0] {
				b.WriteString(plain[prev:loc[0]])
			}
			matchText := plain[loc[0]:loc[1]]
			if isActive {
				b.WriteString(searchActiveStyle.Render(matchText))
			} else {
				b.WriteString(searchHighlightStyle.Render(matchText))
			}
			prev = loc[1]
		}
		if prev < len(plain) {
			b.WriteString(plain[prev:])
		}
		result = append(result, b.String())
	}

	return strings.Join(result, "\n")
}

// StatusView returns the search bar content for the status bar.
func (m SearchModel) StatusView() string {
	switch m.state {
	case SearchInput:
		return promptStyle.Render("/") + " " + m.input.View()
	case SearchNavigate:
		info := fmt.Sprintf("/%s", m.pattern)
		if m.err != nil {
			info += " " + errorStyle.Render(m.err.Error())
		} else if len(m.matches) > 0 {
			info += fmt.Sprintf(" [%d/%d]", m.current+1, len(m.matches))
		} else {
			info += " [no matches]"
		}
		return lipgloss.NewStyle().Foreground(colorWarning).Render(info)
	}
	return ""
}
