package bike

import (
	"bytes"
	"strings"
	"testing"
)

func TestIDGenerator(t *testing.T) {
	g := &IDGenerator{}

	first := g.Next()
	second := g.Next()

	if first == second {
		t.Errorf("IDs should be unique, got %q twice", first)
	}
	if first == "" || second == "" {
		t.Error("IDs should not be empty")
	}

	rootID := g.RootID()
	if len(rootID) < 2 {
		t.Errorf("root ID should be at least 2 chars, got %q", rootID)
	}
}

func TestRenderEmptyDocument(t *testing.T) {
	doc := &Document{
		RootID: "testRoot",
		Rows:   nil,
	}

	var buf bytes.Buffer
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := buf.String()
	assertContains(t, output, `<?xml version="1.0" encoding="UTF-8"?>`)
	assertContains(t, output, `<ul id="testRoot">`)
	assertContains(t, output, `</ul>`)
	assertContains(t, output, `</html>`)
}

func TestRenderBodyRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "r1",
				Type:    RowTypeBody,
				Content: []InlineNode{TextRun{Text: "Hello world"}},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `<li id="r1">`)
	assertContains(t, output, `<p>Hello world</p>`)
	// Body rows should NOT have data-type
	assertNotContains(t, output, `data-type`)
}

func TestRenderHeadingRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "h1",
				Type:    RowTypeHeading,
				Content: []InlineNode{TextRun{Text: "My Heading"}},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `data-type="heading"`)
	assertContains(t, output, `<p>My Heading</p>`)
}

func TestRenderEmptyRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "e1",
				Type:    RowTypeBody,
				Content: nil,
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `<p/>`)
}

func TestRenderSpanWrapping(t *testing.T) {
	// Plain text only — no span
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					TextRun{Text: "Just plain text"},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<p>Just plain text</p>`)
	assertNotContains(t, output, `<span>`)

	// Mixed formatting — text gets span
	doc2 := &Document{
		RootID: "root2",
		Rows: []*Row{
			{
				ID:   "r2",
				Type: RowTypeBody,
				Content: []InlineNode{
					TextRun{Text: "Hello "},
					StrongRun{Children: []InlineNode{TextRun{Text: "world"}}},
				},
			},
		},
	}
	output2 := renderToString(t, doc2)
	assertContains(t, output2, `<span>Hello </span><strong>world</strong>`)
}

func TestRenderInlineFormatting(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					StrongRun{Children: []InlineNode{TextRun{Text: "bold"}}},
					TextRun{Text: " and "},
					EmRun{Children: []InlineNode{TextRun{Text: "italic"}}},
					TextRun{Text: " and "},
					CodeRun{Text: "code"},
					TextRun{Text: " and "},
					LinkRun{URL: "https://example.com", Children: []InlineNode{TextRun{Text: "link"}}},
					TextRun{Text: " and "},
					StrikethroughRun{Children: []InlineNode{TextRun{Text: "strike"}}},
					TextRun{Text: " and "},
					MarkRun{Children: []InlineNode{TextRun{Text: "highlight"}}},
				},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `<strong>bold</strong>`)
	assertContains(t, output, `<em>italic</em>`)
	assertContains(t, output, `<code>code</code>`)
	assertContains(t, output, `<a href="https://example.com">link</a>`)
	assertContains(t, output, `<s>strike</s>`)
	assertContains(t, output, `<mark>highlight</mark>`)
	// TextRuns should be wrapped in <span> because there are formatting nodes
	assertContains(t, output, `<span> and </span>`)
}

func TestRenderNestedRows(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "h1",
				Type:    RowTypeHeading,
				Content: []InlineNode{TextRun{Text: "Parent"}},
				Children: []*Row{
					{
						ID:      "c1",
						Type:    RowTypeBody,
						Content: []InlineNode{TextRun{Text: "Child"}},
					},
				},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `<li id="h1" data-type="heading">`)
	assertContains(t, output, `<p>Parent</p>`)
	assertContains(t, output, `<ul>`)
	assertContains(t, output, `<li id="c1">`)
	assertContains(t, output, `<p>Child</p>`)
}

func TestRenderTaskRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "t1",
				Type:    RowTypeTask,
				Content: []InlineNode{TextRun{Text: "Unchecked"}},
			},
			{
				ID:      "t2",
				Type:    RowTypeTask,
				Content: []InlineNode{TextRun{Text: "Checked"}},
				DoneAt:  "2026-02-16T14:30:47Z",
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `<li id="t1" data-type="task">`)
	assertNotContains(t, output, `id="t1" data-done`)
	assertContains(t, output, `<li id="t2" data-done="2026-02-16T14:30:47Z" data-type="task">`)
}

func TestRenderCodeRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "c1",
				Type:    RowTypeCode,
				Content: []InlineNode{TextRun{Text: "func main() {}"}},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `data-type="code"`)
	assertContains(t, output, `<p>func main() {}</p>`)
}

func TestRenderFormattingOnlyNoSpan(t *testing.T) {
	// Case 2 from CLAUDE.md: <p> contains ONLY formatting (no plain text) → no <span>
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					CodeRun{Text: "some_method"},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<p><code>some_method</code></p>`)
	assertNotContains(t, output, `<span>`)
}

func TestRenderSpanWrappingWhitespaceOnly(t *testing.T) {
	// Whitespace-only text runs between formatting elements get <span>
	// Example: <strong>Size on disk:</strong><span> </span><code>17,792,600,039 bytes</code>
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					StrongRun{Children: []InlineNode{TextRun{Text: "Size on disk:"}}},
					TextRun{Text: " "},
					CodeRun{Text: "17,792,600,039 bytes"},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<strong>Size on disk:</strong><span> </span><code>17,792,600,039 bytes</code>`)
}

func TestRenderSpanWrappingMultipleTextRuns(t *testing.T) {
	// Multiple text runs interspersed with formatting all get <span>
	// Example: <span>Can we disable </span><code>metadata.enabled</code><span> ?</span>
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					TextRun{Text: "Can we disable "},
					CodeRun{Text: "metadata.enabled"},
					TextRun{Text: " ?"},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<span>Can we disable </span><code>metadata.enabled</code><span> ?</span>`)
}

func TestRenderSpanWrappingTrailingText(t *testing.T) {
	// Formatting then text → trailing text gets <span>
	// Example: <strong>Label:</strong><span> trailing text</span>
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					StrongRun{Children: []InlineNode{TextRun{Text: "Label:"}}},
					TextRun{Text: " trailing text"},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<strong>Label:</strong><span> trailing text</span>`)
}

func TestRenderNestedInlineFormatting(t *testing.T) {
	// Nested formatting: <em><mark>text</mark></em>
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:   "r1",
				Type: RowTypeBody,
				Content: []InlineNode{
					EmRun{Children: []InlineNode{
						MarkRun{Children: []InlineNode{TextRun{Text: "highlighted italic"}}},
					}},
					TextRun{Text: " and plain text."},
				},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<em><mark>highlighted italic</mark></em><span> and plain text.</span>`)
}

func TestRenderAttributeOrder(t *testing.T) {
	// Attribute order must be: id, data-done, data-type
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "t1",
				Type:    RowTypeTask,
				Content: []InlineNode{TextRun{Text: "Done task"}},
				DoneAt:  "2026-02-16T14:30:47Z",
			},
		},
	}
	output := renderToString(t, doc)
	// Verify the exact attribute order
	assertContains(t, output, `id="t1" data-done="2026-02-16T14:30:47Z" data-type="task"`)
}

func TestRenderIndentation(t *testing.T) {
	// Verify 2-space indentation at each nesting level
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "h1",
				Type:    RowTypeHeading,
				Content: []InlineNode{TextRun{Text: "Parent"}},
				Children: []*Row{
					{
						ID:      "c1",
						Type:    RowTypeBody,
						Content: []InlineNode{TextRun{Text: "Child"}},
					},
				},
			},
		},
	}
	output := renderToString(t, doc)
	// Root <ul> at 4 spaces (depth 2), rows at 6 spaces (depth 3), nested <ul> at 8, nested rows at 10
	assertContains(t, output, "      <li id=\"h1\" data-type=\"heading\">\n")
	assertContains(t, output, "        <p>Parent</p>\n")
	assertContains(t, output, "        <ul>\n")
	assertContains(t, output, "          <li id=\"c1\">\n")
	assertContains(t, output, "            <p>Child</p>\n")
}

func TestRenderQuoteRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "q1",
				Type:    RowTypeQuote,
				Content: []InlineNode{TextRun{Text: "A quote"}},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<li id="q1" data-type="quote">`)
	assertContains(t, output, `<p>A quote</p>`)
}

func TestRenderOrderedRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "o1",
				Type:    RowTypeOrdered,
				Content: []InlineNode{TextRun{Text: "First"}},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<li id="o1" data-type="ordered">`)
}

func TestRenderUnorderedRow(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "u1",
				Type:    RowTypeUnordered,
				Content: []InlineNode{TextRun{Text: "Item"}},
			},
		},
	}
	output := renderToString(t, doc)
	assertContains(t, output, `<li id="u1" data-type="unordered">`)
}

func TestRenderDocumentStructure(t *testing.T) {
	// Verify the complete document skeleton
	doc := &Document{
		RootID: "testRoot",
		Rows: []*Row{
			{
				ID:      "r1",
				Type:    RowTypeBody,
				Content: []InlineNode{TextRun{Text: "Hello"}},
			},
		},
	}
	output := renderToString(t, doc)

	// Verify exact structure ordering
	lines := strings.Split(output, "\n")
	expected := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<html>`,
		`  <head>`,
		`    <meta charset="utf-8"/>`,
		`  </head>`,
		`  <body>`,
		`    <ul id="testRoot">`,
	}
	for i, want := range expected {
		if i >= len(lines) {
			t.Fatalf("output has fewer lines than expected, missing line %d: %q", i, want)
		}
		if lines[i] != want {
			t.Errorf("line %d:\n  got:  %q\n  want: %q", i, lines[i], want)
		}
	}

	// Verify closing tags are present and in order
	assertContains(t, output, "    </ul>\n  </body>\n</html>\n")
}

func TestRenderHTMLEscaping(t *testing.T) {
	doc := &Document{
		RootID: "root1",
		Rows: []*Row{
			{
				ID:      "r1",
				Type:    RowTypeBody,
				Content: []InlineNode{TextRun{Text: `<script>alert("xss")</script>`}},
			},
		},
	}

	output := renderToString(t, doc)
	assertContains(t, output, `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`)
	assertNotContains(t, output, `<script>`)
}

func renderToString(t *testing.T, doc *Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
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
