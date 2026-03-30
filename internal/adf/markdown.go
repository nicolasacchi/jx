package adf

import (
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
