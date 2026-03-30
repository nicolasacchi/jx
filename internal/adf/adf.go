package adf

import "encoding/json"

// Node represents an ADF document node.
type Node map[string]any

// Document represents a complete ADF document.
type Document struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
	Content []Node `json:"content"`
}

// Builder provides a fluent API for constructing ADF documents.
type Builder struct {
	nodes []Node
}

// New creates a new ADF document builder.
func New() *Builder {
	return &Builder{}
}

// Heading adds a heading node (level 1-6).
func (b *Builder) Heading(level int, text string) *Builder {
	b.nodes = append(b.nodes, Node{
		"type":  "heading",
		"attrs": map[string]any{"level": level},
		"content": []Node{
			{"type": "text", "text": text},
		},
	})
	return b
}

// Paragraph adds a paragraph with plain text.
func (b *Builder) Paragraph(text string) *Builder {
	if text == "" {
		return b
	}
	b.nodes = append(b.nodes, Node{
		"type": "paragraph",
		"content": []Node{
			{"type": "text", "text": text},
		},
	})
	return b
}

// ParagraphWithMarks adds a paragraph with inline text nodes (supports marks).
func (b *Builder) ParagraphWithMarks(inlines []Node) *Builder {
	if len(inlines) == 0 {
		return b
	}
	b.nodes = append(b.nodes, Node{
		"type":    "paragraph",
		"content": inlines,
	})
	return b
}

// CodeBlock adds a code block with language.
func (b *Builder) CodeBlock(language, text string) *Builder {
	node := Node{
		"type": "codeBlock",
		"content": []Node{
			{"type": "text", "text": text},
		},
	}
	if language != "" {
		node["attrs"] = map[string]any{"language": language}
	}
	return b.appendNode(node)
}

// BulletList adds a bullet list from string items.
func (b *Builder) BulletList(items ...string) *Builder {
	if len(items) == 0 {
		return b
	}
	listItems := make([]Node, len(items))
	for i, item := range items {
		listItems[i] = Node{
			"type": "listItem",
			"content": []Node{
				{
					"type": "paragraph",
					"content": []Node{
						{"type": "text", "text": item},
					},
				},
			},
		}
	}
	return b.appendNode(Node{
		"type":    "bulletList",
		"content": listItems,
	})
}

// BulletListNodes adds a bullet list from pre-built list item nodes.
func (b *Builder) BulletListNodes(items []Node) *Builder {
	if len(items) == 0 {
		return b
	}
	return b.appendNode(Node{
		"type":    "bulletList",
		"content": items,
	})
}

// OrderedList adds an ordered list from string items.
func (b *Builder) OrderedList(items ...string) *Builder {
	if len(items) == 0 {
		return b
	}
	listItems := make([]Node, len(items))
	for i, item := range items {
		listItems[i] = Node{
			"type": "listItem",
			"content": []Node{
				{
					"type": "paragraph",
					"content": []Node{
						{"type": "text", "text": item},
					},
				},
			},
		}
	}
	return b.appendNode(Node{
		"type":    "orderedList",
		"content": listItems,
	})
}

// Rule adds a horizontal rule.
func (b *Builder) Rule() *Builder {
	return b.appendNode(Node{"type": "rule"})
}

// Blockquote adds a blockquote with a paragraph.
func (b *Builder) Blockquote(text string) *Builder {
	return b.appendNode(Node{
		"type": "blockquote",
		"content": []Node{
			{
				"type": "paragraph",
				"content": []Node{
					{"type": "text", "text": text},
				},
			},
		},
	})
}

// RawNode adds a pre-built node directly.
func (b *Builder) RawNode(node Node) *Builder {
	return b.appendNode(node)
}

func (b *Builder) appendNode(node Node) *Builder {
	b.nodes = append(b.nodes, node)
	return b
}

// Build constructs the final ADF document.
func (b *Builder) Build() *Document {
	if len(b.nodes) == 0 {
		b.nodes = []Node{}
	}
	return &Document{
		Version: 1,
		Type:    "doc",
		Content: b.nodes,
	}
}

// BuildJSON constructs the ADF document and marshals to JSON.
func (b *Builder) BuildJSON() (json.RawMessage, error) {
	doc := b.Build()
	return json.Marshal(doc)
}

// --- Inline node constructors ---

// Text creates a plain text inline node.
func Text(text string) Node {
	return Node{"type": "text", "text": text}
}

// Bold creates a bold text inline node.
func Bold(text string) Node {
	return Node{
		"type":  "text",
		"text":  text,
		"marks": []Node{{"type": "strong"}},
	}
}

// Italic creates an italic text inline node.
func Italic(text string) Node {
	return Node{
		"type":  "text",
		"text":  text,
		"marks": []Node{{"type": "em"}},
	}
}

// Code creates an inline code text node.
func Code(text string) Node {
	return Node{
		"type":  "text",
		"text":  text,
		"marks": []Node{{"type": "code"}},
	}
}

// Link creates a linked text node.
func Link(text, href string) Node {
	return Node{
		"type": "text",
		"text": text,
		"marks": []Node{
			{"type": "link", "attrs": map[string]any{"href": href}},
		},
	}
}

// ListItem creates a list item node from inline nodes.
func ListItem(inlines ...Node) Node {
	return Node{
		"type": "listItem",
		"content": []Node{
			{"type": "paragraph", "content": inlines},
		},
	}
}
