package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msjurset/golars"
)

type Mode int

const (
	ModeTable Mode = iota
	ModeStats
	ModeFilter
	ModeSQL
	ModeColInfo
)

var modeNames = []string{"Table", "Stats", "Filter", "SQL", "Columns"}

// Messages
type FilterAppliedMsg struct{ DF *golars.DataFrame }
type FilterClearedMsg struct{}
type SQLResultMsg struct{ DF *golars.DataFrame }
type ErrorMsg struct{ Err error }

type Model struct {
	originalDF *golars.DataFrame
	currentDF  *golars.DataFrame
	fileName   string
	mode       Mode
	width      int
	height     int
	showHelp   bool

	// Sub-models
	tableView   TableModel
	statsView   StatsModel
	filterView  FilterModel
	sqlView     SQLModel
	colInfoView ColInfoModel
	exportView  ExportModel
	searchView  SearchModel

	filterText string
	err        error
}

func NewModel(df *golars.DataFrame, fileName string) Model {
	em := NewExportModel()
	em.SetDataFrame(df)
	return Model{
		originalDF:  df,
		currentDF:   df,
		fileName:    fileName,
		mode:        ModeTable,
		tableView:   NewTableModel(df),
		statsView:   NewStatsModel(df),
		filterView:  NewFilterModel(df),
		sqlView:     NewSQLModel(df, fileName),
		colInfoView: NewColInfoModel(df),
		exportView:  em,
		searchView:  NewSearchModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		contentHeight := m.height - 4 // tabs + status bar + borders
		m.tableView.SetSize(m.width, contentHeight)
		m.statsView.SetSize(m.width, contentHeight)
		m.filterView.SetSize(m.width, contentHeight)
		m.sqlView.SetSize(m.width, contentHeight)
		m.colInfoView.SetSize(m.width, contentHeight)
		m.searchView.width = m.width
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Export overlay handles its own keys
		if m.exportView.Active() {
			var cmd tea.Cmd
			m.exportView, cmd = m.exportView.Update(msg)
			return m, cmd
		}

		// Search: open with / when not in text input or detail view
		if msg.String() == "/" && !m.isInputActive() && !m.tableView.showDetail {
			if m.searchView.state == SearchNavigate {
				// Re-open input from navigate state
				m.searchView.state = SearchInput
				m.searchView.input.SetValue(m.searchView.pattern)
				m.searchView.input.Focus()
				return m, textinput.Blink
			}
			if m.searchView.state == SearchInactive {
				cmd := m.searchView.Open()
				return m, cmd
			}
		}

		// Search input state: all keys go to the search text input
		if m.searchView.state == SearchInput {
			var cmd tea.Cmd
			m.searchView, cmd = m.searchView.Update(msg)
			// If search just transitioned to Navigate, scan data and jump to first match
			if m.searchView.state == SearchNavigate {
				m.searchView.ScanDataFrame(m.displayedDF())
				m.goToCurrentMatch()
			}
			return m, cmd
		}

		// Search navigate state: handle n/N/esc, let everything else fall through
		if m.searchView.state == SearchNavigate {
			switch msg.String() {
			case "n":
				m.searchView.NextMatch()
				m.goToCurrentMatch()
				return m, nil
			case "N":
				m.searchView.PrevMatch()
				m.goToCurrentMatch()
				return m, nil
			case "esc":
				m.searchView.Close()
				return m, nil
			}
			// All other keys fall through to normal handling
		}

		// Export trigger — works from any view when not in text input
		if msg.String() == "ctrl+e" && !m.isInputActive() {
			m.exportView.SetDataFrame(m.currentDF)
			m.exportView.width = m.width
			cmd := m.exportView.Open()
			return m, cmd
		}

		// Help toggle — works everywhere
		if msg.String() == "?" && !m.isInputActive() {
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Dismiss help on any key
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Mode switching (only when not in text input or detail view)
		if !m.isInputActive() && !m.tableView.showDetail {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "1":
				m.mode = ModeTable
				return m, nil
			case "2":
				m.mode = ModeStats
				return m, nil
			case "3":
				m.mode = ModeFilter
				m.filterView.Focus()
				return m, textinput.Blink
			case "4":
				m.mode = ModeSQL
				m.sqlView.Focus()
				return m, textinput.Blink
			case "5":
				m.mode = ModeColInfo
				return m, nil
			case "tab":
				m.mode = (m.mode + 1) % Mode(len(modeNames))
				if m.mode == ModeFilter {
					m.filterView.Focus()
					return m, textinput.Blink
				}
				if m.mode == ModeSQL {
					m.sqlView.Focus()
					return m, textinput.Blink
				}
				return m, nil
			case "shift+tab":
				m.mode = (m.mode + Mode(len(modeNames)) - 1) % Mode(len(modeNames))
				if m.mode == ModeFilter {
					m.filterView.Focus()
					return m, textinput.Blink
				}
				if m.mode == ModeSQL {
					m.sqlView.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}
		}

	case FilterAppliedMsg:
		m.currentDF = msg.DF
		m.filterText = m.filterView.input.Value()
		m.err = nil
		m.tableView.SetDataFrame(msg.DF)
		m.statsView.SetDataFrame(msg.DF)
		m.colInfoView.SetDataFrame(msg.DF)
		m.sqlView.SetDataFrame(msg.DF, m.fileName)
		m.mode = ModeTable
		return m, nil

	case FilterClearedMsg:
		m.currentDF = m.originalDF
		m.filterText = ""
		m.err = nil
		m.tableView.SetDataFrame(m.originalDF)
		m.statsView.SetDataFrame(m.originalDF)
		m.colInfoView.SetDataFrame(m.originalDF)
		m.sqlView.SetDataFrame(m.originalDF, m.fileName)
		m.mode = ModeTable
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	// Delegate to active sub-model
	var cmd tea.Cmd
	switch m.mode {
	case ModeTable:
		m.tableView, cmd = m.tableView.Update(msg)
	case ModeStats:
		m.statsView, cmd = m.statsView.Update(msg)
	case ModeFilter:
		m.filterView, cmd = m.filterView.Update(msg)
	case ModeSQL:
		m.sqlView, cmd = m.sqlView.Update(msg)
	case ModeColInfo:
		m.colInfoView, cmd = m.colInfoView.Update(msg)
	}
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Tab bar
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	if m.exportView.Active() {
		b.WriteString(m.exportView.View())
	} else if m.showHelp {
		b.WriteString(m.renderHelp())
	} else {
		// Active view content
		var viewContent string
		switch m.mode {
		case ModeTable:
			viewContent = m.tableView.View()
		case ModeStats:
			viewContent = m.statsView.View()
		case ModeFilter:
			viewContent = m.filterView.View()
		case ModeSQL:
			viewContent = m.sqlView.View()
		case ModeColInfo:
			viewContent = m.colInfoView.View()
		}
		if m.searchView.Active() {
			activeLine := m.activeMatchLine()
			viewContent = m.searchView.HighlightContent(viewContent, activeLine)
		}
		b.WriteString(viewContent)
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m Model) renderTabs() string {
	var tabs []string
	for i, name := range modeNames {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		if Mode(i) == m.mode {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, tabStyle.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().Width(m.width).Render(row)
}

func (m Model) renderStatusBar() string {
	h, w := m.currentDF.Shape()

	left := fmt.Sprintf(" %s │ %d rows × %d cols", m.fileName, h, w)
	if m.filterText != "" {
		left += fmt.Sprintf(" │ filter: %s", m.filterText)
	}

	var right string
	if m.searchView.Active() {
		right = " " + m.searchView.StatusView() + " "
	} else {
		right = " q:quit  tab:switch  ctrl+e:export  /:search  ?:help "
	}

	leftRendered := statusBarStyle.Render(left)
	rightRendered := statusBarStyle.Render(right)
	gap := m.width - lipgloss.Width(leftRendered) - lipgloss.Width(rightRendered)
	if gap < 0 {
		gap = 0
	}
	mid := statusBarStyle.Render(strings.Repeat(" ", gap))

	return leftRendered + mid + rightRendered
}

func (m Model) renderHelp() string {
	title := statHeaderStyle.Render("  Keyboard Shortcuts")

	sections := []struct {
		header string
		keys   [][2]string
	}{
		{
			header: "Navigation",
			keys: [][2]string{
				{"1-5", "Switch to view: Table, Stats, Filter, SQL, Columns"},
				{"tab / shift+tab", "Cycle through views"},
				{"ctrl+e", "Export current data to file"},
				{"q / ctrl+c", "Quit"},
				{"?", "Toggle this help"},
			},
		},
		{
			header: "Table View",
			keys: [][2]string{
				{"j/k  or  up/down", "Move cursor up/down"},
				{"h/l  or  left/right", "Scroll columns left/right"},
				{"enter", "Open row detail view"},
				{"pgup / pgdn", "Page up/down"},
				{"ctrl+f / ctrl+b", "Page down/up"},
				{"ctrl+d / ctrl+u / J / K", "Half-page down/up"},
				{"space", "Toggle row detail view"},
				{"g / G", "Jump to first/last page"},
				{"s", "Sort by current column (asc -> desc -> none)"},
				{"S", "Clear sort"},
				{"w", "Toggle float display (wide/compact)"},
			},
		},
		{
			header: "Stats & Columns Views",
			keys: [][2]string{
				{"j/k  or  up/down", "Scroll / select column"},
				{"pgup / pgdn", "Jump by several entries"},
				{"g", "Jump to top"},
			},
		},
		{
			header: "Filter View",
			keys: [][2]string{
				{"enter", "Apply filter expression"},
				{"ctrl+r", "Clear filter (restore full dataset)"},
				{"up / down", "Browse filter history"},
				{"esc", "Unfocus input"},
			},
		},
		{
			header: "SQL View",
			keys: [][2]string{
				{"enter", "Execute SQL query"},
				{"ctrl+l", "Clear result"},
				{"up / down", "Browse query history"},
				{"esc", "Unfocus input"},
			},
		},
		{
			header: "Search",
			keys: [][2]string{
				{"/", "Open search (regex)"},
				{"n", "Next match"},
				{"N", "Previous match"},
				{"esc", "Dismiss search"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(title + "\n\n")

	for _, sec := range sections {
		b.WriteString(infoStyle.Render("  "+sec.header) + "\n")
		for _, kv := range sec.keys {
			key := statusKeyStyle.Render(fmt.Sprintf("    %-24s", kv[0]))
			desc := helpStyle.Render(kv[1])
			b.WriteString(key + desc + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  Press any key to dismiss"))

	return lipgloss.NewStyle().Width(m.width).Render(b.String())
}

// displayedDF returns the DataFrame currently being displayed, accounting for
// sort order in the table view.
func (m Model) displayedDF() *golars.DataFrame {
	switch m.mode {
	case ModeTable:
		return m.tableView.sortedDF
	case ModeSQL:
		if m.sqlView.result != nil {
			return m.sqlView.table.sortedDF
		}
	}
	return m.currentDF
}

// activeMatchLine returns the rendered line index where the current search
// match should be highlighted with the active (red) style, or -1 if unknown.
func (m Model) activeMatchLine() int {
	row, _ := m.searchView.CurrentMatch()
	if row < 0 {
		return -1
	}
	switch m.mode {
	case ModeTable:
		// Table: 3 header lines (header, types, separator), then data rows.
		// Data row i corresponds to absolute row page*pageSize + i.
		pageStart := m.tableView.page * m.tableView.pageSize
		localRow := row - pageStart
		if localRow < 0 || localRow >= m.tableView.pageSize {
			return -1 // not on current page
		}
		return 3 + localRow
	case ModeSQL:
		if m.sqlView.result != nil {
			pageStart := m.sqlView.table.page * m.sqlView.table.pageSize
			localRow := row - pageStart
			if localRow < 0 || localRow >= m.sqlView.table.pageSize {
				return -1
			}
			// SQL view has input line(s) then table with same 3-line header
			// This is approximate; the exact offset depends on SQL view layout
			return -1 // too complex to map reliably, skip active highlight for SQL
		}
		return -1
	default:
		// Stats, Filter, ColInfo don't have a simple row-to-line mapping
		return -1
	}
}

// goToCurrentMatch scrolls the active view to show the current search match.
func (m *Model) goToCurrentMatch() {
	row, col := m.searchView.CurrentMatch()
	if row < 0 {
		return
	}
	switch m.mode {
	case ModeTable:
		m.tableView.GoToRow(row)
		m.tableView.GoToCol(col)
	case ModeStats:
		m.statsView.GoToCol(col)
	case ModeFilter:
		// Filter view doesn't have scrollable data rows
	case ModeSQL:
		if m.sqlView.result != nil {
			m.sqlView.table.GoToRow(row)
			m.sqlView.table.GoToCol(col)
		}
	case ModeColInfo:
		m.colInfoView.GoToCol(col)
	}
}

func (m Model) isInputActive() bool {
	if m.searchView.state == SearchInput {
		return true
	}
	switch m.mode {
	case ModeFilter:
		return m.filterView.input.Focused()
	case ModeSQL:
		return m.sqlView.input.Focused()
	}
	return false
}
