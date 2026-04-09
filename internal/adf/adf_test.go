package adf

import (
	"encoding/json"
	"testing"
)

func TestBuilderBasic(t *testing.T) {
	doc := New().
		Heading(2, "Title").
		Paragraph("Hello world").
		Build()

	if doc.Version != 1 {
		t.Errorf("version = %d, want 1", doc.Version)
	}
	if doc.Type != "doc" {
		t.Errorf("type = %q, want doc", doc.Type)
	}
	if len(doc.Content) != 2 {
		t.Fatalf("content length = %d, want 2", len(doc.Content))
	}
	if doc.Content[0]["type"] != "heading" {
		t.Errorf("first node type = %v, want heading", doc.Content[0]["type"])
	}
}

func TestBuilderCodeBlock(t *testing.T) {
	doc := New().
		CodeBlock("ruby", "def show\n  @product = Product.find(params[:id])\nend").
		Build()

	if len(doc.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(doc.Content))
	}
	node := doc.Content[0]
	if node["type"] != "codeBlock" {
		t.Errorf("type = %v, want codeBlock", node["type"])
	}
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		t.Fatal("attrs missing")
	}
	if attrs["language"] != "ruby" {
		t.Errorf("language = %v, want ruby", attrs["language"])
	}
}

func TestBuilderBulletList(t *testing.T) {
	doc := New().BulletList("First", "Second").Build()
	if len(doc.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(doc.Content))
	}
	list := doc.Content[0]
	if list["type"] != "bulletList" {
		t.Errorf("type = %v, want bulletList", list["type"])
	}
	items, ok := list["content"].([]Node)
	if !ok {
		t.Fatal("content is not []Node")
	}
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}
}

func TestBuildJSON(t *testing.T) {
	data, err := New().Paragraph("test").BuildJSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if doc.Version != 1 {
		t.Errorf("version = %d", doc.Version)
	}
}

func TestInlineNodes(t *testing.T) {
	bold := Bold("important")
	if bold["type"] != "text" {
		t.Error("bold type wrong")
	}
	marks, ok := bold["marks"].([]Node)
	if !ok || len(marks) != 1 || marks[0]["type"] != "strong" {
		t.Error("bold marks wrong")
	}

	italic := Italic("note")
	marks2, ok := italic["marks"].([]Node)
	if !ok || len(marks2) != 1 || marks2[0]["type"] != "em" {
		t.Error("italic marks wrong")
	}

	code := Code("func")
	marks3, ok := code["marks"].([]Node)
	if !ok || len(marks3) != 1 || marks3[0]["type"] != "code" {
		t.Error("code marks wrong")
	}

	link := Link("click", "https://example.com")
	marks4, ok := link["marks"].([]Node)
	if !ok || len(marks4) != 1 {
		t.Error("link marks wrong")
	}
}

// --- Markdown converter tests ---

func TestMarkdownHeadings(t *testing.T) {
	doc := FromMarkdown("# Title\n\n## Subtitle\n\n### H3")
	if len(doc.Content) != 3 {
		t.Fatalf("content = %d, want 3", len(doc.Content))
	}
	for _, node := range doc.Content {
		if node["type"] != "heading" {
			t.Errorf("type = %v, want heading", node["type"])
		}
	}
}

func TestMarkdownCodeBlock(t *testing.T) {
	md := "# Context\n\n```ruby\ndef show\n  @product = Product.find(params[:id])\nend\n```\n\nDone."
	doc := FromMarkdown(md)

	// Should be: heading, codeBlock, paragraph
	if len(doc.Content) != 3 {
		t.Fatalf("content = %d, want 3 (got: %v)", len(doc.Content), nodeTypes(doc.Content))
	}
	if doc.Content[0]["type"] != "heading" {
		t.Errorf("[0] type = %v, want heading", doc.Content[0]["type"])
	}
	if doc.Content[1]["type"] != "codeBlock" {
		t.Errorf("[1] type = %v, want codeBlock", doc.Content[1]["type"])
	}
	// Verify language attr
	attrs, _ := doc.Content[1]["attrs"].(map[string]any)
	if attrs["language"] != "ruby" {
		t.Errorf("language = %v, want ruby", attrs["language"])
	}
	// Verify code content
	content, _ := doc.Content[1]["content"].([]Node)
	if len(content) != 1 {
		t.Fatalf("code content = %d", len(content))
	}
	code := content[0]["text"].(string)
	if code != "def show\n  @product = Product.find(params[:id])\nend" {
		t.Errorf("code = %q", code)
	}
}

func TestMarkdownBulletList(t *testing.T) {
	md := "- First item\n- Second item\n- Third item"
	doc := FromMarkdown(md)
	if len(doc.Content) != 1 {
		t.Fatalf("content = %d, want 1", len(doc.Content))
	}
	if doc.Content[0]["type"] != "bulletList" {
		t.Errorf("type = %v, want bulletList", doc.Content[0]["type"])
	}
}

func TestMarkdownOrderedList(t *testing.T) {
	md := "1. First\n2. Second\n3. Third"
	doc := FromMarkdown(md)
	if len(doc.Content) != 1 {
		t.Fatalf("content = %d, want 1", len(doc.Content))
	}
	if doc.Content[0]["type"] != "orderedList" {
		t.Errorf("type = %v, want orderedList", doc.Content[0]["type"])
	}
}

func TestMarkdownInlineMarks(t *testing.T) {
	md := "This is **bold** and *italic* and `code` text."
	doc := FromMarkdown(md)
	if len(doc.Content) != 1 {
		t.Fatalf("content = %d, want 1", len(doc.Content))
	}
	para := doc.Content[0]
	content, _ := para["content"].([]Node)
	// Should have: "This is ", bold, " and ", italic, " and ", code, " text."
	if len(content) != 7 {
		t.Fatalf("inline nodes = %d, want 7", len(content))
	}
	// Check bold
	if content[1]["text"] != "bold" {
		t.Errorf("bold text = %v", content[1]["text"])
	}
	marks, _ := content[1]["marks"].([]Node)
	if len(marks) != 1 || marks[0]["type"] != "strong" {
		t.Error("bold marks wrong")
	}
}

func TestMarkdownLink(t *testing.T) {
	md := "See [PR #6966](https://github.com/1000farmacie/1000farmacie/pull/6966) for details."
	doc := FromMarkdown(md)
	if len(doc.Content) != 1 {
		t.Fatalf("content = %d", len(doc.Content))
	}
	content, _ := doc.Content[0]["content"].([]Node)
	// Should have: "See ", link, " for details."
	if len(content) != 3 {
		t.Fatalf("inline nodes = %d, want 3", len(content))
	}
	if content[1]["text"] != "PR #6966" {
		t.Errorf("link text = %v", content[1]["text"])
	}
}

func TestMarkdownMixed(t *testing.T) {
	md := `## Context

Product page caching Phase 1a is deployed.

## Task

- Modify ProductsController#show
- Set s-maxage=0

## Verification

` + "```bash\ncurl -I https://www.1000farmacie.it/prodotto/...\n```"

	doc := FromMarkdown(md)
	types := nodeTypes(doc.Content)
	// heading, paragraph, heading, bulletList, heading, codeBlock
	expected := []string{"heading", "paragraph", "heading", "bulletList", "heading", "codeBlock"}
	if len(types) != len(expected) {
		t.Fatalf("types = %v, want %v", types, expected)
	}
	for i, typ := range types {
		if typ != expected[i] {
			t.Errorf("[%d] = %s, want %s", i, typ, expected[i])
		}
	}
}

func TestMarkdownValidJSON(t *testing.T) {
	md := "## Title\n\n```go\nfunc main() {}\n```\n\n- item 1\n- item 2"
	doc := FromMarkdown(md)
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	// Verify it round-trips as valid JSON
	var check map[string]any
	if err := json.Unmarshal(data, &check); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
}

// --- ToMarkdown tests ---

func TestToMarkdownHeading(t *testing.T) {
	doc := New().Heading(2, "My Title").Build()
	got := ToMarkdown(doc)
	if got != "## My Title" {
		t.Errorf("got %q, want %q", got, "## My Title")
	}
}

func TestToMarkdownParagraph(t *testing.T) {
	doc := New().Paragraph("Hello world").Build()
	got := ToMarkdown(doc)
	if got != "Hello world" {
		t.Errorf("got %q, want %q", got, "Hello world")
	}
}

func TestToMarkdownCodeBlock(t *testing.T) {
	doc := New().CodeBlock("ruby", "def show\n  @product\nend").Build()
	got := ToMarkdown(doc)
	expected := "```ruby\ndef show\n  @product\nend\n```"
	if got != expected {
		t.Errorf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestToMarkdownBulletList(t *testing.T) {
	doc := New().BulletList("First", "Second", "Third").Build()
	got := ToMarkdown(doc)
	expected := "- First\n- Second\n- Third"
	if got != expected {
		t.Errorf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestToMarkdownOrderedList(t *testing.T) {
	doc := New().OrderedList("First", "Second").Build()
	got := ToMarkdown(doc)
	expected := "1. First\n2. Second"
	if got != expected {
		t.Errorf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestToMarkdownRule(t *testing.T) {
	doc := New().Paragraph("Above").Rule().Paragraph("Below").Build()
	got := ToMarkdown(doc)
	expected := "Above\n\n---\n\nBelow"
	if got != expected {
		t.Errorf("got:\n%s\nwant:\n%s", got, expected)
	}
}

func TestToMarkdownBlockquote(t *testing.T) {
	doc := New().Blockquote("A quote").Build()
	got := ToMarkdown(doc)
	expected := "> A quote"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestToMarkdownInlineMarks(t *testing.T) {
	doc := New().Build()
	doc.Content = []Node{
		{
			"type": "paragraph",
			"content": []Node{
				Text("Hello "),
				Bold("world"),
				Text(" and "),
				Italic("italic"),
				Text(" and "),
				Code("code"),
			},
		},
	}
	got := ToMarkdown(doc)
	expected := "Hello **world** and *italic* and `code`"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestToMarkdownLink(t *testing.T) {
	doc := New().Build()
	doc.Content = []Node{
		{
			"type": "paragraph",
			"content": []Node{
				Text("See "),
				Link("PR #6966", "https://github.com/pull/6966"),
			},
		},
	}
	got := ToMarkdown(doc)
	expected := "See [PR #6966](https://github.com/pull/6966)"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestToMarkdownMixed(t *testing.T) {
	doc := New().
		Heading(2, "Context").
		Paragraph("Phase 1a is deployed.").
		Heading(2, "Task").
		BulletList("Modify controller", "Set s-maxage=0").
		Heading(2, "Code").
		CodeBlock("bash", "curl -I https://example.com").
		Build()
	got := ToMarkdown(doc)
	expected := `## Context

Phase 1a is deployed.

## Task

- Modify controller
- Set s-maxage=0

## Code

` + "```bash\ncurl -I https://example.com\n```"
	if got != expected {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, expected)
	}
}

func TestToMarkdownFromJSON(t *testing.T) {
	adfJSON := `{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Hello from ADF"}]}]}`
	got := ToMarkdownFromJSON(json.RawMessage(adfJSON))
	if got != "Hello from ADF" {
		t.Errorf("got %q, want %q", got, "Hello from ADF")
	}
}

func TestToMarkdownFromJSONPlainString(t *testing.T) {
	got := ToMarkdownFromJSON(json.RawMessage(`"just a string"`))
	if got != "just a string" {
		t.Errorf("got %q, want %q", got, "just a string")
	}
}

func TestToMarkdownFromJSONNil(t *testing.T) {
	got := ToMarkdownFromJSON(nil)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestToMarkdownNilDoc(t *testing.T) {
	got := ToMarkdown(nil)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestToMarkdownRoundTrip(t *testing.T) {
	// Build an ADF doc via FromMarkdown, convert back, verify similarity
	original := "## Context\n\nProduct page caching is deployed.\n\n- Modify controller\n- Set headers\n\n```bash\ncurl -I https://example.com\n```"
	doc := FromMarkdown(original)
	got := ToMarkdown(doc)
	if got != original {
		t.Errorf("round-trip mismatch:\ngot:\n%s\n\nwant:\n%s", got, original)
	}
}

func TestToMarkdownTable(t *testing.T) {
	doc := &Document{
		Version: 1,
		Type:    "doc",
		Content: []Node{
			{
				"type": "table",
				"content": []Node{
					{
						"type": "tableRow",
						"content": []Node{
							{"type": "tableHeader", "content": []Node{{"type": "paragraph", "content": []Node{Text("Name")}}}},
							{"type": "tableHeader", "content": []Node{{"type": "paragraph", "content": []Node{Text("Value")}}}},
						},
					},
					{
						"type": "tableRow",
						"content": []Node{
							{"type": "tableCell", "content": []Node{{"type": "paragraph", "content": []Node{Text("CPU")}}}},
							{"type": "tableCell", "content": []Node{{"type": "paragraph", "content": []Node{Text("80%")}}}},
						},
					},
				},
			},
		},
	}
	got := ToMarkdown(doc)
	expected := "| Name | Value |\n| --- | --- |\n| CPU | 80% |"
	if got != expected {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, expected)
	}
}

func nodeTypes(nodes []Node) []string {
	types := make([]string, len(nodes))
	for i, n := range nodes {
		if t, ok := n["type"].(string); ok {
			types[i] = t
		}
	}
	return types
}
