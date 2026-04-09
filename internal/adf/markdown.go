package adf

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FromMarkdown converts a markdown string to an ADF document.
// Supports: headings (#-######), code blocks (```), bullet lists (-/*), ordered lists (1.),
// bold (**), italic (*), inline code (`), links [text](url), blockquotes (>), horizontal rules (---).
func FromMarkdown(md string) *Document {
	b := New()
	lines := strings.Split(md, "\n")
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Fenced code block
		if strings.HasPrefix(line, "```") {
			lang := strings.TrimPrefix(line, "```")
			lang = strings.TrimSpace(lang)
			var codeLines []string
			i++
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			if i < len(lines) {
				i++ // skip closing ```
			}
			b.CodeBlock(lang, strings.Join(codeLines, "\n"))
			continue
		}

		// Heading
		if level, text := parseHeading(line); level > 0 {
			b.Heading(level, text)
			i++
			continue
		}

		// Horizontal rule
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			b.Rule()
			i++
			continue
		}

		// Blockquote
		if strings.HasPrefix(trimmed, "> ") {
			text := strings.TrimPrefix(trimmed, "> ")
			b.Blockquote(text)
			i++
			continue
		}

		// Bullet list
		if isBulletItem(trimmed) {
			var items []Node
			for i < len(lines) && isBulletItem(strings.TrimSpace(lines[i])) {
				text := strings.TrimSpace(lines[i])
				text = text[2:] // strip "- " or "* "
				inlines := parseInlineMarks(text)
				items = append(items, ListItem(inlines...))
				i++
			}
			b.BulletListNodes(items)
			continue
		}

		// Ordered list
		if isOrderedItem(trimmed) {
			var items []Node
			for i < len(lines) && isOrderedItem(strings.TrimSpace(lines[i])) {
				text := strings.TrimSpace(lines[i])
				idx := strings.Index(text, ". ")
				if idx >= 0 {
					text = text[idx+2:]
				}
				inlines := parseInlineMarks(text)
				items = append(items, ListItem(inlines...))
				i++
			}
			b.appendNode(Node{
				"type":    "orderedList",
				"content": items,
			})
			continue
		}

		// Empty line — skip
		if trimmed == "" {
			i++
			continue
		}

		// Regular paragraph — collect contiguous non-empty, non-special lines
		var paraLines []string
		for i < len(lines) {
			l := strings.TrimSpace(lines[i])
			if l == "" || strings.HasPrefix(lines[i], "```") || parseHeadingLevel(l) > 0 ||
				isBulletItem(l) || isOrderedItem(l) ||
				l == "---" || l == "***" || l == "___" ||
				strings.HasPrefix(l, "> ") {
				break
			}
			paraLines = append(paraLines, lines[i])
			i++
		}
		if len(paraLines) > 0 {
			text := strings.Join(paraLines, " ")
			inlines := parseInlineMarks(text)
			b.ParagraphWithMarks(inlines)
		}
	}

	return b.Build()
}

func parseHeading(line string) (int, string) {
	level := parseHeadingLevel(strings.TrimSpace(line))
	if level == 0 {
		return 0, ""
	}
	text := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
	return level, text
}

func parseHeadingLevel(line string) int {
	level := 0
	for _, c := range line {
		if c == '#' {
			level++
		} else {
			break
		}
	}
	if level > 0 && level <= 6 && len(line) > level && line[level] == ' ' {
		return level
	}
	return 0
}

func isBulletItem(line string) bool {
	return strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
}

func isOrderedItem(line string) bool {
	for i, c := range line {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '.' && i > 0 && i+1 < len(line) && line[i+1] == ' ' {
			return true
		}
		return false
	}
	return false
}

// ToMarkdown converts an ADF document to markdown.
// Handles: heading, paragraph, codeBlock, bulletList, orderedList, blockquote, rule,
// table, mediaSingle/media, and inline marks (strong, em, code, link).
// Unknown node types are handled via recursive text extraction.
func ToMarkdown(doc *Document) string {
	if doc == nil || len(doc.Content) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, node := range doc.Content {
		if i > 0 {
			sb.WriteString("\n")
		}
		renderNode(&sb, node, "")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ToMarkdownFromJSON parses raw ADF JSON and converts to markdown.
// Returns empty string if the input is nil, empty, or not valid ADF.
func ToMarkdownFromJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		// Not valid ADF — try to return as plain string
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return s
		}
		return ""
	}
	if doc.Type != "doc" {
		return ""
	}
	return ToMarkdown(&doc)
}

func renderNode(sb *strings.Builder, node Node, indent string) {
	typ, _ := node["type"].(string)
	switch typ {
	case "heading":
		level := getAttrInt(node, "level", 1)
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		renderInlineContent(sb, node)
		sb.WriteString("\n")
	case "paragraph":
		sb.WriteString(indent)
		renderInlineContent(sb, node)
		sb.WriteString("\n")
	case "codeBlock":
		lang := getAttrString(node, "language")
		sb.WriteString("```")
		sb.WriteString(lang)
		sb.WriteString("\n")
		renderInlineContent(sb, node)
		sb.WriteString("\n```\n")
	case "bulletList":
		renderList(sb, node, indent, false)
	case "orderedList":
		renderList(sb, node, indent, true)
	case "blockquote":
		children := getContent(node)
		for _, child := range children {
			childType, _ := child["type"].(string)
			if childType == "paragraph" {
				sb.WriteString("> ")
				renderInlineContent(sb, child)
				sb.WriteString("\n")
			} else {
				sb.WriteString("> ")
				renderNode(sb, child, "> ")
			}
		}
	case "rule":
		sb.WriteString("---\n")
	case "table":
		renderTable(sb, node)
	case "mediaSingle":
		children := getContent(node)
		for _, child := range children {
			renderNode(sb, child, indent)
		}
	case "media":
		alt := getAttrString(node, "alt")
		if alt == "" {
			alt = "attachment"
		}
		sb.WriteString(fmt.Sprintf("[%s]", alt))
		sb.WriteString("\n")
	default:
		// Unknown block node — try to extract text recursively
		children := getContent(node)
		if len(children) > 0 {
			for _, child := range children {
				renderNode(sb, child, indent)
			}
		} else {
			text, _ := node["text"].(string)
			if text != "" {
				sb.WriteString(indent)
				sb.WriteString(text)
				sb.WriteString("\n")
			}
		}
	}
}

func renderInlineContent(sb *strings.Builder, node Node) {
	children := getContent(node)
	for _, child := range children {
		typ, _ := child["type"].(string)
		if typ == "text" {
			text, _ := child["text"].(string)
			sb.WriteString(applyMarks(text, child))
		} else if typ == "hardBreak" {
			sb.WriteString("\n")
		} else if typ == "mention" {
			text, _ := child["attrs"].(map[string]any)["text"].(string)
			if text != "" {
				sb.WriteString(text)
			}
		} else if typ == "inlineCard" {
			url := getAttrString(child, "url")
			if url != "" {
				sb.WriteString(url)
			}
		} else {
			// Recurse for unknown inline types
			text, _ := child["text"].(string)
			if text != "" {
				sb.WriteString(text)
			} else {
				renderInlineContent(sb, child)
			}
		}
	}
}

func applyMarks(text string, node Node) string {
	marks := getMarks(node)
	if len(marks) == 0 {
		return text
	}
	result := text
	for _, mark := range marks {
		markType, _ := mark["type"].(string)
		switch markType {
		case "strong":
			result = "**" + result + "**"
		case "em":
			result = "*" + result + "*"
		case "code":
			result = "`" + result + "`"
		case "link":
			href := ""
			if attrs, ok := mark["attrs"].(map[string]any); ok {
				href, _ = attrs["href"].(string)
			}
			if href != "" {
				result = "[" + result + "](" + href + ")"
			}
		case "strike":
			result = "~~" + result + "~~"
		}
	}
	return result
}

func renderList(sb *strings.Builder, node Node, indent string, ordered bool) {
	items := getContent(node)
	for i, item := range items {
		children := getContent(item)
		prefix := "- "
		if ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}
		for j, child := range children {
			childType, _ := child["type"].(string)
			if j == 0 {
				sb.WriteString(indent)
				sb.WriteString(prefix)
				if childType == "paragraph" {
					renderInlineContent(sb, child)
					sb.WriteString("\n")
				} else {
					// Nested list or other block
					renderNode(sb, child, indent+"  ")
				}
			} else {
				// Continuation content (nested list under a list item)
				renderNode(sb, child, indent+"  ")
			}
		}
	}
}

func renderTable(sb *strings.Builder, node Node) {
	rows := getContent(node)
	if len(rows) == 0 {
		return
	}

	// Collect all rows as string slices
	var table [][]string
	for _, row := range rows {
		cells := getContent(row)
		var rowData []string
		for _, cell := range cells {
			var cellSB strings.Builder
			children := getContent(cell)
			for _, child := range children {
				renderInlineContent(&cellSB, child)
			}
			rowData = append(rowData, strings.TrimSpace(cellSB.String()))
		}
		table = append(table, rowData)
	}

	if len(table) == 0 {
		return
	}

	// Find max columns
	maxCols := 0
	for _, row := range table {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Render header row
	renderTableRow(sb, table[0], maxCols)
	sb.WriteString("\n")

	// Separator
	sb.WriteString("|")
	for i := 0; i < maxCols; i++ {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range table[1:] {
		renderTableRow(sb, row, maxCols)
		sb.WriteString("\n")
	}
}

func renderTableRow(sb *strings.Builder, row []string, maxCols int) {
	for i := 0; i < maxCols; i++ {
		sb.WriteString("| ")
		if i < len(row) {
			sb.WriteString(row[i])
		}
		sb.WriteString(" ")
	}
	sb.WriteString("|")
}

func getContent(node Node) []Node {
	raw, ok := node["content"]
	if !ok {
		return nil
	}
	// Handle both []Node and []any (from JSON unmarshaling)
	if nodes, ok := raw.([]Node); ok {
		return nodes
	}
	if arr, ok := raw.([]any); ok {
		nodes := make([]Node, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				nodes = append(nodes, Node(m))
			}
		}
		return nodes
	}
	return nil
}

func getMarks(node Node) []Node {
	raw, ok := node["marks"]
	if !ok {
		return nil
	}
	if nodes, ok := raw.([]Node); ok {
		return nodes
	}
	if arr, ok := raw.([]any); ok {
		nodes := make([]Node, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				nodes = append(nodes, Node(m))
			}
		}
		return nodes
	}
	return nil
}

func getAttrString(node Node, key string) string {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return ""
	}
	s, _ := attrs[key].(string)
	return s
}

func getAttrInt(node Node, key string, defaultVal int) int {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return defaultVal
	}
	// JSON unmarshal produces float64 for numbers
	if f, ok := attrs[key].(float64); ok {
		return int(f)
	}
	if i, ok := attrs[key].(int); ok {
		return i
	}
	return defaultVal
}

// parseInlineMarks parses inline markdown marks: **bold**, *italic*, `code`, [text](url).
func parseInlineMarks(text string) []Node {
	var nodes []Node
	i := 0
	buf := strings.Builder{}

	flush := func() {
		if buf.Len() > 0 {
			nodes = append(nodes, Text(buf.String()))
			buf.Reset()
		}
	}

	for i < len(text) {
		// Bold: **text**
		if i+1 < len(text) && text[i] == '*' && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "**")
			if end >= 0 {
				flush()
				nodes = append(nodes, Bold(text[i+2:i+2+end]))
				i = i + 2 + end + 2
				continue
			}
		}

		// Italic: *text* (but not **)
		if text[i] == '*' && (i+1 >= len(text) || text[i+1] != '*') {
			end := strings.Index(text[i+1:], "*")
			if end >= 0 && end > 0 {
				flush()
				nodes = append(nodes, Italic(text[i+1:i+1+end]))
				i = i + 1 + end + 1
				continue
			}
		}

		// Inline code: `text`
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end >= 0 {
				flush()
				nodes = append(nodes, Code(text[i+1:i+1+end]))
				i = i + 1 + end + 1
				continue
			}
		}

		// Link: [text](url)
		if text[i] == '[' {
			closeBracket := strings.Index(text[i:], "](")
			if closeBracket >= 0 {
				closeAbs := i + closeBracket
				closeParen := strings.Index(text[closeAbs+2:], ")")
				if closeParen >= 0 {
					linkText := text[i+1 : closeAbs]
					linkURL := text[closeAbs+2 : closeAbs+2+closeParen]
					flush()
					nodes = append(nodes, Link(linkText, linkURL))
					i = closeAbs + 2 + closeParen + 1
					continue
				}
			}
		}

		buf.WriteByte(text[i])
		i++
	}

	flush()
	return nodes
}
