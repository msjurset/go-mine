package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// JSONNode represents a node in a JSON tree for interactive viewing.
type JSONNode struct {
	Key       string      // object key (empty for array elements / root)
	Value     interface{} // raw value for leaf nodes
	Children  []*JSONNode // child nodes for objects/arrays
	Parent    *JSONNode   // parent node (nil for root)
	NodeType  jsonNodeType
	Collapsed bool
	Depth     int
}

type jsonNodeType int

const (
	jsonObject jsonNodeType = iota
	jsonArray
	jsonString
	jsonNumber
	jsonBool
	jsonNull
)

// JSONTreeView is a scrollable, collapsible JSON tree viewer with syntax highlighting.
type JSONTreeView struct {
	root      *JSONNode
	flatLines []flatLine // pre-rendered visible lines
	scrollY   int
	cursorY   int // which line the cursor is on (for collapse/expand)
	width     int
	height    int
	dirty     bool // needs re-flattening
	truncated bool
}

type flatLine struct {
	node    *JSONNode
	text    string // rendered text for this line
	indent  int
	isOpen  bool // opening brace/bracket line (collapsible)
	isClose bool // closing brace/bracket line
}

func NewJSONTreeView() JSONTreeView {
	return JSONTreeView{dirty: true}
}

// SetData parses JSON results into the tree.
func (v *JSONTreeView) SetData(results []interface{}, truncated bool) {
	v.truncated = truncated
	var root interface{}
	if len(results) == 1 {
		root = results[0]
	} else {
		root = results
	}
	v.root = buildNode("", root, 0)
	v.scrollY = 0
	v.cursorY = 0
	v.dirty = true
}

// SetSize updates the viewport dimensions.
func (v *JSONTreeView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

// Toggle collapses or expands the node at the cursor (walks up to parent if on a leaf).
func (v *JSONTreeView) Toggle() {
	v.ensureFlat()
	if n := v.cursorNode(); n != nil {
		n.Collapsed = !n.Collapsed
		v.dirty = true
		if n.Collapsed {
			v.moveCursorToNode(n)
		}
	}
}

// ExpandAll recursively expands the node at the cursor and all its descendants.
func (v *JSONTreeView) ExpandAll() {
	v.ensureFlat()
	if n := v.cursorNode(); n != nil {
		setCollapsedRecursive(n, false)
		v.dirty = true
	}
}

// CollapseAll recursively collapses the node at the cursor and all its descendants.
func (v *JSONTreeView) CollapseAll() {
	v.ensureFlat()
	if n := v.cursorNode(); n != nil {
		setCollapsedRecursive(n, true)
		v.dirty = true
		v.moveCursorToNode(n)
	}
}

// moveCursorToNode repositions the cursor to the line belonging to the given node
// after a re-flatten.
func (v *JSONTreeView) moveCursorToNode(target *JSONNode) {
	v.ensureFlat()
	for i, fl := range v.flatLines {
		if fl.node == target {
			v.cursorY = i
			v.ensureVisible()
			return
		}
	}
}

// cursorNode returns the nearest collapsible node at or owning the current cursor line.
// If the cursor is on a leaf, walks up to the nearest container parent.
func (v *JSONTreeView) cursorNode() *JSONNode {
	if v.cursorY < 0 || v.cursorY >= len(v.flatLines) {
		return nil
	}
	n := v.flatLines[v.cursorY].node
	for n != nil {
		if (n.NodeType == jsonObject || n.NodeType == jsonArray) && len(n.Children) > 0 {
			return n
		}
		n = n.Parent
	}
	return nil
}

func setCollapsedRecursive(n *JSONNode, collapsed bool) {
	if n.NodeType == jsonObject || n.NodeType == jsonArray {
		if len(n.Children) > 0 {
			n.Collapsed = collapsed
		}
		for _, c := range n.Children {
			setCollapsedRecursive(c, collapsed)
		}
	}
}

// CursorDown moves cursor down.
func (v *JSONTreeView) CursorDown() {
	v.ensureFlat()
	if v.cursorY < len(v.flatLines)-1 {
		v.cursorY++
	}
	v.ensureVisible()
}

// CursorUp moves cursor up.
func (v *JSONTreeView) CursorUp() {
	if v.cursorY > 0 {
		v.cursorY--
	}
	v.ensureVisible()
}

// PageDown scrolls down by a page.
func (v *JSONTreeView) PageDown() {
	v.ensureFlat()
	vis := v.visibleHeight()
	v.cursorY += vis
	if v.cursorY >= len(v.flatLines) {
		v.cursorY = len(v.flatLines) - 1
	}
	v.ensureVisible()
}

// PageUp scrolls up by a page.
func (v *JSONTreeView) PageUp() {
	vis := v.visibleHeight()
	v.cursorY -= vis
	if v.cursorY < 0 {
		v.cursorY = 0
	}
	v.ensureVisible()
}

// GoToTop moves to the first line.
func (v *JSONTreeView) GoToTop() {
	v.cursorY = 0
	v.scrollY = 0
}

// GoToBottom moves to the last line.
func (v *JSONTreeView) GoToBottom() {
	v.ensureFlat()
	v.cursorY = len(v.flatLines) - 1
	if v.cursorY < 0 {
		v.cursorY = 0
	}
	v.ensureVisible()
}

func (v *JSONTreeView) ensureVisible() {
	vis := v.visibleHeight()
	if v.cursorY < v.scrollY {
		v.scrollY = v.cursorY
	}
	if v.cursorY >= v.scrollY+vis {
		v.scrollY = v.cursorY - vis + 1
	}
	if v.scrollY < 0 {
		v.scrollY = 0
	}
}

func (v *JSONTreeView) visibleHeight() int {
	h := v.height - 2 // room for scroll indicator
	if h < 1 {
		h = 1
	}
	return h
}

// HasData returns whether there's any data to display.
func (v *JSONTreeView) HasData() bool {
	return v.root != nil
}

// LineCount returns the total number of visible lines.
func (v *JSONTreeView) LineCount() int {
	v.ensureFlat()
	return len(v.flatLines)
}

// View renders the tree.
func (v *JSONTreeView) View() string {
	if v.root == nil {
		return ""
	}

	v.ensureFlat()

	vis := v.visibleHeight()
	total := len(v.flatLines)

	start := v.scrollY
	end := start + vis
	if end > total {
		end = total
		start = end - vis
		if start < 0 {
			start = 0
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		fl := v.flatLines[i]
		prefix := "  "
		if i == v.cursorY {
			prefix = jsonCursorStyle.Render("▸ ")
		}
		b.WriteString(prefix + fl.text + "\n")
	}

	if v.truncated {
		b.WriteString(jsonTruncStyle.Render(fmt.Sprintf("  ... (truncated at %d results)", maxJQResults)) + "\n")
	}

	// Scroll indicator
	if total > vis {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  [line %d/%d]  ", v.cursorY+1, total)))
		b.WriteString(helpStyle.Render("j/k:navigate  enter/space:toggle  E:expand tree  C:collapse tree"))
	}

	return b.String()
}

// ensureFlat rebuilds the flat line list if dirty.
func (v *JSONTreeView) ensureFlat() {
	if !v.dirty {
		return
	}
	v.flatLines = nil
	if v.root != nil {
		v.flatLines = flattenNode(v.root, "", false)
	}
	v.dirty = false
}

// flattenNode recursively converts the tree into displayable lines.
func flattenNode(n *JSONNode, trailingComma string, isLast bool) []flatLine {
	comma := ","
	if isLast {
		comma = ""
	}
	// Override if caller specified
	if trailingComma != "" {
		comma = trailingComma
	}

	indent := strings.Repeat("  ", n.Depth)
	keyPrefix := ""
	if n.Key != "" {
		keyPrefix = jsonKeyStyle.Render("\""+n.Key+"\"") + jsonPuncStyle.Render(": ")
	}

	switch n.NodeType {
	case jsonObject:
		if len(n.Children) == 0 {
			return []flatLine{{
				node: n,
				text: indent + keyPrefix + jsonPuncStyle.Render("{}") + jsonPuncStyle.Render(comma),
			}}
		}
		if n.Collapsed {
			count := countDescendants(n)
			return []flatLine{{
				node:   n,
				text:   indent + keyPrefix + jsonCollapsedStyle.Render("▶ ") + jsonPuncStyle.Render("{") + jsonCollapsedStyle.Render(fmt.Sprintf(" %d items ", count)) + jsonPuncStyle.Render("}") + jsonPuncStyle.Render(comma),
				isOpen: true,
			}}
		}
		var lines []flatLine
		lines = append(lines, flatLine{
			node:   n,
			text:   indent + keyPrefix + jsonExpandedStyle.Render("▼ ") + jsonPuncStyle.Render("{"),
			isOpen: true,
		})
		for i, child := range n.Children {
			childIsLast := i == len(n.Children)-1
			lines = append(lines, flattenNode(child, "", childIsLast)...)
		}
		lines = append(lines, flatLine{
			node:    n,
			text:    indent + jsonPuncStyle.Render("}") + jsonPuncStyle.Render(comma),
			isClose: true,
		})
		return lines

	case jsonArray:
		if len(n.Children) == 0 {
			return []flatLine{{
				node: n,
				text: indent + keyPrefix + jsonPuncStyle.Render("[]") + jsonPuncStyle.Render(comma),
			}}
		}
		if n.Collapsed {
			count := len(n.Children)
			return []flatLine{{
				node:   n,
				text:   indent + keyPrefix + jsonCollapsedStyle.Render("▶ ") + jsonPuncStyle.Render("[") + jsonCollapsedStyle.Render(fmt.Sprintf(" %d items ", count)) + jsonPuncStyle.Render("]") + jsonPuncStyle.Render(comma),
				isOpen: true,
			}}
		}
		var lines []flatLine
		lines = append(lines, flatLine{
			node:   n,
			text:   indent + keyPrefix + jsonExpandedStyle.Render("▼ ") + jsonPuncStyle.Render("["),
			isOpen: true,
		})
		for i, child := range n.Children {
			childIsLast := i == len(n.Children)-1
			lines = append(lines, flattenNode(child, "", childIsLast)...)
		}
		lines = append(lines, flatLine{
			node:    n,
			text:    indent + jsonPuncStyle.Render("]") + jsonPuncStyle.Render(comma),
			isClose: true,
		})
		return lines

	case jsonString:
		val := formatJSONString(n.Value)
		return []flatLine{{
			node: n,
			text: indent + keyPrefix + jsonStringStyle.Render(val) + jsonPuncStyle.Render(comma),
		}}

	case jsonNumber:
		val := fmt.Sprintf("%v", n.Value)
		return []flatLine{{
			node: n,
			text: indent + keyPrefix + jsonNumberStyle.Render(val) + jsonPuncStyle.Render(comma),
		}}

	case jsonBool:
		val := "false"
		if b, ok := n.Value.(bool); ok && b {
			val = "true"
		}
		return []flatLine{{
			node: n,
			text: indent + keyPrefix + jsonBoolStyle.Render(val) + jsonPuncStyle.Render(comma),
		}}

	case jsonNull:
		return []flatLine{{
			node: n,
			text: indent + keyPrefix + jsonNullStyle.Render("null") + jsonPuncStyle.Render(comma),
		}}
	}

	return nil
}

func formatJSONString(v interface{}) string {
	s, ok := v.(string)
	if !ok {
		return `""`
	}
	// Use json.Marshal to properly escape the string
	b, err := json.Marshal(s)
	if err != nil {
		return `"` + s + `"`
	}
	return string(b)
}

func countDescendants(n *JSONNode) int {
	count := len(n.Children)
	for _, c := range n.Children {
		if c.NodeType == jsonObject || c.NodeType == jsonArray {
			count += countDescendants(c)
		}
	}
	return count
}

// buildNode constructs the tree recursively from arbitrary Go values.
func buildNode(key string, val interface{}, depth int) *JSONNode {
	if val == nil {
		return &JSONNode{Key: key, NodeType: jsonNull, Depth: depth}
	}

	switch v := val.(type) {
	case map[string]interface{}:
		node := &JSONNode{Key: key, NodeType: jsonObject, Depth: depth}
		// Sort keys for deterministic order
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			child := buildNode(k, v[k], depth+1)
			child.Parent = node
			node.Children = append(node.Children, child)
		}
		return node

	case []interface{}:
		node := &JSONNode{Key: key, NodeType: jsonArray, Depth: depth}
		for _, item := range v {
			child := buildNode("", item, depth+1)
			child.Parent = node
			node.Children = append(node.Children, child)
		}
		return node

	case string:
		return &JSONNode{Key: key, Value: v, NodeType: jsonString, Depth: depth}

	case float64:
		return &JSONNode{Key: key, Value: v, NodeType: jsonNumber, Depth: depth}

	case int:
		return &JSONNode{Key: key, Value: v, NodeType: jsonNumber, Depth: depth}

	case int64:
		return &JSONNode{Key: key, Value: v, NodeType: jsonNumber, Depth: depth}

	case bool:
		return &JSONNode{Key: key, Value: v, NodeType: jsonBool, Depth: depth}

	default:
		// Fallback: marshal to JSON string
		b, err := json.Marshal(v)
		if err != nil {
			return &JSONNode{Key: key, Value: fmt.Sprintf("%v", v), NodeType: jsonString, Depth: depth}
		}
		return &JSONNode{Key: key, Value: string(b), NodeType: jsonString, Depth: depth}
	}
}

// --- Styles for JSON syntax highlighting ---

var (
	jsonKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")) // blue

	jsonStringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399")) // green

	jsonNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")) // amber

	jsonBoolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F472B6")) // pink

	jsonNullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")). // gray
			Italic(true)

	jsonPuncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")) // light gray for {, }, [, ], :, ,

	jsonCollapsedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8B5CF6")). // purple
				Italic(true)

	jsonExpandedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")) // subtle gray for ▼

	jsonCursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")). // amber cursor
			Bold(true)

	jsonTruncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Italic(true)

	// Autocomplete styles
	acSelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#4C1D95")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	acItemStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#D1D5DB"))

	acBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6B7280")).
			Background(lipgloss.Color("#1F2937"))

	acScrollStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
)
