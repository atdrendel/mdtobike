package convert

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/atdrendel/mdtobike/internal/bike"
)

// fixedTime returns a fixed clock for deterministic test output.
func fixedTime() time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-03-01T12:00:00Z")
	return t
}

func convertAndRender(t *testing.T, markdown string) string {
	t.Helper()
	doc, err := fromMarkdown([]byte(markdown), fixedTime)
	if err != nil {
		t.Fatalf("fromMarkdown() error = %v", err)
	}
	var buf bytes.Buffer
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}

func TestEmptyInput(t *testing.T) {
	output := convertAndRender(t, "")
	assertContains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assertContains(t, output, `<html>`)
	assertContains(t, output, `</html>`)
}

func TestHeading(t *testing.T) {
	output := convertAndRender(t, "# Hello World")
	assertContains(t, output, `data-type="heading"`)
	assertContains(t, output, `<p>Hello World</p>`)
}

func TestHeadingHierarchy(t *testing.T) {
	md := `# Top Level
## Section A
Paragraph under A
### Subsection
## Section B
# Another Top`

	output := convertAndRender(t, md)

	// Top Level should contain Section A and Section B as children
	assertContains(t, output, `<p>Top Level</p>`)
	assertContains(t, output, `<p>Section A</p>`)
	assertContains(t, output, `<p>Paragraph under A</p>`)
	assertContains(t, output, `<p>Subsection</p>`)
	assertContains(t, output, `<p>Section B</p>`)
	assertContains(t, output, `<p>Another Top</p>`)

	// Verify nesting structure: Section A should be inside Top Level's <ul>
	// and Subsection inside Section A's <ul>
	doc, _ := fromMarkdown([]byte(md), fixedTime)
	if len(doc.Rows) != 2 {
		t.Fatalf("expected 2 top-level rows, got %d", len(doc.Rows))
	}
	topLevel := doc.Rows[0]
	topText := inlineText(topLevel.Content)
	if topText != "Top Level" {
		t.Errorf("expected 'Top Level', got %q", topText)
	}
	if len(topLevel.Children) != 2 {
		t.Fatalf("expected 2 children under Top Level, got %d", len(topLevel.Children))
	}
	sectionA := topLevel.Children[0]
	saText := inlineText(sectionA.Content)
	if saText != "Section A" {
		t.Errorf("expected 'Section A', got %q", saText)
	}
	if len(sectionA.Children) != 2 {
		t.Fatalf("expected 2 children under Section A (paragraph + subsection), got %d", len(sectionA.Children))
	}
}

func TestParagraph(t *testing.T) {
	output := convertAndRender(t, "Hello world")
	assertContains(t, output, `<p>Hello world</p>`)
	assertNotContains(t, output, `data-type`)
}

func TestBoldAndItalic(t *testing.T) {
	output := convertAndRender(t, "**bold** and *italic*")
	assertContains(t, output, `<strong>bold</strong>`)
	assertContains(t, output, `<em>italic</em>`)
}

func TestInlineCode(t *testing.T) {
	output := convertAndRender(t, "Use `fmt.Println`")
	assertContains(t, output, `<code>fmt.Println</code>`)
}

func TestLink(t *testing.T) {
	output := convertAndRender(t, "[Example](https://example.com)")
	assertContains(t, output, `<a href="https://example.com">Example</a>`)
}

func TestStrikethrough(t *testing.T) {
	output := convertAndRender(t, "~~deleted~~")
	assertContains(t, output, `<s>deleted</s>`)
}

func TestHighlight(t *testing.T) {
	output := convertAndRender(t, "==highlighted==")
	assertContains(t, output, `<mark>highlighted</mark>`)
}

func TestFencedCodeBlock(t *testing.T) {
	md := "```go\nfunc main() {\n  fmt.Println(\"hello\")\n}\n```"
	output := convertAndRender(t, md)
	// Should produce one code row per line, no language info
	assertContains(t, output, `data-type="code"`)
	assertContains(t, output, `<p>func main() {</p>`)
	assertContains(t, output, `<p>  fmt.Println(&#34;hello&#34;)</p>`) // quotes get HTML-escaped
	assertContains(t, output, `<p>}</p>`)
}

func TestBlockquote(t *testing.T) {
	md := "> First line\n> Second line"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="quote"`)
}

func TestUnorderedList(t *testing.T) {
	md := "- One\n- Two\n- Three"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="unordered"`)
	assertContains(t, output, `<p>One</p>`)
	assertContains(t, output, `<p>Two</p>`)
	assertContains(t, output, `<p>Three</p>`)
}

func TestOrderedList(t *testing.T) {
	md := "1. First\n2. Second\n3. Third"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="ordered"`)
	assertContains(t, output, `<p>First</p>`)
}

func TestNestedList(t *testing.T) {
	md := "- Parent\n  - Child 1\n  - Child 2"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 1 {
		t.Fatalf("expected 1 top-level row, got %d", len(doc.Rows))
	}
	parent := doc.Rows[0]
	if parent.Type != bike.RowTypeUnordered {
		t.Errorf("expected unordered, got %s", parent.Type)
	}
	if len(parent.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(parent.Children))
	}
}

func TestTaskListUnchecked(t *testing.T) {
	md := "- [ ] Todo item"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="task"`)
	assertNotContains(t, output, `data-done`)
}

func TestTaskListChecked(t *testing.T) {
	md := "- [x] Done item"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="task"`)
	assertContains(t, output, `data-done="2026-03-01T12:00:00Z"`)
}

func TestThematicBreak(t *testing.T) {
	md := "Before\n\n---\n\nAfter"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="hr"`)
	assertContains(t, output, `<p/>`)
	assertNotContains(t, output, `<p>---</p>`)
}

func TestImage(t *testing.T) {
	md := "![Alt text](https://example.com/img.png)"
	output := convertAndRender(t, md)
	assertContains(t, output, `![Alt text](https://example.com/img.png)`)
}

func TestContentBeforeHeading(t *testing.T) {
	md := "Some intro text\n\n# First Heading"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 2 {
		t.Fatalf("expected 2 top-level rows, got %d", len(doc.Rows))
	}
	if doc.Rows[0].Type != bike.RowTypeBody {
		t.Errorf("first row should be body, got %s", doc.Rows[0].Type)
	}
	if doc.Rows[1].Type != bike.RowTypeHeading {
		t.Errorf("second row should be heading, got %s", doc.Rows[1].Type)
	}
}

func TestMixedDocument(t *testing.T) {
	md := `# Project

A description paragraph.

## Tasks

- [x] Setup
- [ ] Implementation

## Notes

> Important note

` + "```" + `
code here
` + "```" + `

---

**Bold** and *italic* text.`

	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="heading"`)
	assertContains(t, output, `data-type="task"`)
	assertContains(t, output, `data-type="quote"`)
	assertContains(t, output, `data-type="code"`)
	assertContains(t, output, `data-type="hr"`)
	assertContains(t, output, `<strong>Bold</strong>`)
	assertContains(t, output, `<em>italic</em>`)
}

func TestNestedInlineFormatting(t *testing.T) {
	// *==highlighted italic==* should produce <em><mark>...</mark></em>
	output := convertAndRender(t, "*==highlighted italic==*")
	assertContains(t, output, `<em><mark>highlighted italic</mark></em>`)
}

func TestBlockquoteMultipleRows(t *testing.T) {
	// Each paragraph in a blockquote becomes a separate quote row (siblings)
	md := "> First line\n>\n> Second line"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	quoteCount := 0
	for _, row := range doc.Rows {
		if row.Type == bike.RowTypeQuote {
			quoteCount++
		}
	}
	if quoteCount != 2 {
		t.Errorf("expected 2 quote rows, got %d", quoteCount)
	}
}

func TestBlockquoteWithInlineFormatting(t *testing.T) {
	md := "> **bold** and *italic* text"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="quote"`)
	assertContains(t, output, `<strong>bold</strong>`)
	assertContains(t, output, `<em>italic</em>`)
}

func TestTaskWithInlineFormatting(t *testing.T) {
	// Task items with mixed inline content should use span wrapping
	md := "- [x] Did we replace `some_method`"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="task"`)
	assertContains(t, output, `<code>some_method</code>`)
	// "Did we replace " should be in a <span> since there's also a <code> element
	assertContains(t, output, `<span>Did we replace </span>`)
}

func TestListItemWithInlineFormatting(t *testing.T) {
	md := "- **bold** item"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="unordered"`)
	assertContains(t, output, `<strong>bold</strong>`)
	assertContains(t, output, `<span> item</span>`)
}

func TestOrderedListAllItems(t *testing.T) {
	// Verify all items get data-type="ordered"
	md := "1. First\n2. Second\n3. Third"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(doc.Rows))
	}
	for i, row := range doc.Rows {
		if row.Type != bike.RowTypeOrdered {
			t.Errorf("row %d: expected ordered, got %s", i, row.Type)
		}
	}
}

func TestCodeBlockWithoutLanguage(t *testing.T) {
	md := "```\nline one\nline two\n```"
	output := convertAndRender(t, md)
	assertContains(t, output, `data-type="code"`)
	assertContains(t, output, `<p>line one</p>`)
	assertContains(t, output, `<p>line two</p>`)
}

func TestCodeBlockWithEmptyLines(t *testing.T) {
	md := "```\nfirst\n\nlast\n```"
	output := convertAndRender(t, md)
	assertContains(t, output, `<p>first</p>`)
	assertContains(t, output, `<p>last</p>`)

	// Verify the empty line becomes an empty code row
	doc, _ := fromMarkdown([]byte(md), fixedTime)
	codeRows := 0
	emptyCodeRows := 0
	for _, row := range doc.Rows {
		if row.Type == bike.RowTypeCode {
			codeRows++
			if len(row.Content) == 0 || (len(row.Content) == 1 && row.Content[0].(bike.TextRun).Text == "") {
				emptyCodeRows++
			}
		}
	}
	if codeRows != 3 {
		t.Errorf("expected 3 code rows, got %d", codeRows)
	}
	if emptyCodeRows != 1 {
		t.Errorf("expected 1 empty code row, got %d", emptyCodeRows)
	}
}

func TestHeadingH2WithoutH1(t *testing.T) {
	// An h2 without a preceding h1 should still be a top-level row
	md := "## Orphan Heading\n\nSome text"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 1 {
		t.Fatalf("expected 1 top-level row, got %d", len(doc.Rows))
	}
	if doc.Rows[0].Type != bike.RowTypeHeading {
		t.Errorf("expected heading, got %s", doc.Rows[0].Type)
	}
	// "Some text" should be a child of the h2
	if len(doc.Rows[0].Children) != 1 {
		t.Fatalf("expected 1 child under h2, got %d", len(doc.Rows[0].Children))
	}
}

func TestHeadingSkippedLevels(t *testing.T) {
	// h1 → h3 (skipping h2) should nest h3 under h1
	md := "# Top\n### Deep"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 1 {
		t.Fatalf("expected 1 top-level row, got %d", len(doc.Rows))
	}
	top := doc.Rows[0]
	if len(top.Children) != 1 {
		t.Fatalf("expected 1 child under Top, got %d", len(top.Children))
	}
	deep := top.Children[0]
	deepText := inlineText(deep.Content)
	if deepText != "Deep" {
		t.Errorf("expected 'Deep', got %q", deepText)
	}
}

func TestMultipleParagraphsUnderHeading(t *testing.T) {
	md := "# Section\n\nFirst paragraph.\n\nSecond paragraph."
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 1 {
		t.Fatalf("expected 1 top-level row, got %d", len(doc.Rows))
	}
	section := doc.Rows[0]
	if len(section.Children) != 2 {
		t.Fatalf("expected 2 children (paragraphs) under heading, got %d", len(section.Children))
	}
	for _, child := range section.Children {
		if child.Type != bike.RowTypeBody {
			t.Errorf("expected body row, got %s", child.Type)
		}
	}
}

func TestSpanWrappingEndToEnd(t *testing.T) {
	// Plain text + formatting → text runs get <span>
	output := convertAndRender(t, "Hello **world**")
	assertContains(t, output, `<span>Hello </span><strong>world</strong>`)
}

func TestFormattingOnlyNoSpan(t *testing.T) {
	// Only formatting, no plain text → no <span>
	output := convertAndRender(t, "`some_method`")
	assertContains(t, output, `<p><code>some_method</code></p>`)
	assertNotContains(t, output, `<span>`)
}

func TestAutolink(t *testing.T) {
	// GFM autolinks
	output := convertAndRender(t, "Visit https://example.com for more")
	assertContains(t, output, `<a href="https://example.com">https://example.com</a>`)
}

func TestCodeBlockSiblingRows(t *testing.T) {
	// Code block lines must be sibling rows, NOT nested
	md := "```\nline1\nline2\nline3\n```"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 3 {
		t.Fatalf("expected 3 top-level code rows, got %d", len(doc.Rows))
	}
	for i, row := range doc.Rows {
		if row.Type != bike.RowTypeCode {
			t.Errorf("row %d: expected code, got %s", i, row.Type)
		}
		if len(row.Children) != 0 {
			t.Errorf("row %d: code rows should not have children, got %d", i, len(row.Children))
		}
	}
}

func TestBlockquoteSiblingRows(t *testing.T) {
	// Blockquote rows should be siblings (not nested under each other)
	md := "> Line one\n>\n> Line two"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	for _, row := range doc.Rows {
		if row.Type != bike.RowTypeQuote {
			t.Errorf("expected quote row, got %s", row.Type)
		}
		if len(row.Children) != 0 {
			t.Errorf("blockquote rows should not have children, got %d", len(row.Children))
		}
	}
}

func TestNestedOrderedList(t *testing.T) {
	md := "1. Parent\n   1. Child"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 1 {
		t.Fatalf("expected 1 top-level row, got %d", len(doc.Rows))
	}
	parent := doc.Rows[0]
	if parent.Type != bike.RowTypeOrdered {
		t.Errorf("expected ordered, got %s", parent.Type)
	}
	if len(parent.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(parent.Children))
	}
	if parent.Children[0].Type != bike.RowTypeOrdered {
		t.Errorf("child expected ordered, got %s", parent.Children[0].Type)
	}
}

func TestMixedListAndHeadings(t *testing.T) {
	// Lists under a heading should be children of that heading
	md := "# Shopping\n\n- Milk\n- Bread\n\n# Notes\n\n1. First note"
	doc, _ := fromMarkdown([]byte(md), fixedTime)

	if len(doc.Rows) != 2 {
		t.Fatalf("expected 2 top-level rows, got %d", len(doc.Rows))
	}

	shopping := doc.Rows[0]
	if inlineText(shopping.Content) != "Shopping" {
		t.Errorf("expected 'Shopping', got %q", inlineText(shopping.Content))
	}
	// 2 unordered list items under Shopping
	unorderedCount := 0
	for _, child := range shopping.Children {
		if child.Type == bike.RowTypeUnordered {
			unorderedCount++
		}
	}
	if unorderedCount != 2 {
		t.Errorf("expected 2 unordered items under Shopping, got %d", unorderedCount)
	}

	notes := doc.Rows[1]
	if inlineText(notes.Content) != "Notes" {
		t.Errorf("expected 'Notes', got %q", inlineText(notes.Content))
	}
	orderedCount := 0
	for _, child := range notes.Children {
		if child.Type == bike.RowTypeOrdered {
			orderedCount++
		}
	}
	if orderedCount != 1 {
		t.Errorf("expected 1 ordered item under Notes, got %d", orderedCount)
	}
}

// inlineText concatenates all TextRun content from a slice of InlineNodes.
func inlineText(nodes []bike.InlineNode) string {
	var sb strings.Builder
	for _, n := range nodes {
		if tr, ok := n.(bike.TextRun); ok {
			sb.WriteString(tr.Text)
		}
	}
	return sb.String()
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("output does not contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("output should not contain %q\ngot:\n%s", substr, s)
	}
}
