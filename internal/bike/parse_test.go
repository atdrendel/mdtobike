package bike

import (
	"strings"
	"testing"
)

// bikeDoc wraps row XML in a minimal valid Bike document.
func bikeDoc(rows string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<html>
  <head>
    <meta charset="utf-8"/>
  </head>
  <body>
    <ul id="testroot">
` + rows + `
    </ul>
  </body>
</html>`
}

func TestParseEmptyDocument(t *testing.T) {
	doc, err := Parse(strings.NewReader(bikeDoc("")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if doc.RootID != "testroot" {
		t.Errorf("RootID = %q, want %q", doc.RootID, "testroot")
	}
	if len(doc.Rows) != 0 {
		t.Errorf("Rows = %d, want 0", len(doc.Rows))
	}
}

func TestParseBodyRow(t *testing.T) {
	input := bikeDoc(`      <li id="a1">
        <p>Hello world</p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(doc.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(doc.Rows))
	}
	row := doc.Rows[0]
	if row.ID != "a1" {
		t.Errorf("ID = %q, want %q", row.ID, "a1")
	}
	if row.Type != RowTypeBody {
		t.Errorf("Type = %q, want %q", row.Type, RowTypeBody)
	}
	if len(row.Content) != 1 {
		t.Fatalf("Content = %d nodes, want 1", len(row.Content))
	}
	tr, ok := row.Content[0].(TextRun)
	if !ok {
		t.Fatalf("Content[0] type = %T, want TextRun", row.Content[0])
	}
	if tr.Text != "Hello world" {
		t.Errorf("Text = %q, want %q", tr.Text, "Hello world")
	}
}

func TestParseRowTypes(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		wantType RowType
	}{
		{"heading", `data-type="heading"`, RowTypeHeading},
		{"quote", `data-type="quote"`, RowTypeQuote},
		{"code", `data-type="code"`, RowTypeCode},
		{"ordered", `data-type="ordered"`, RowTypeOrdered},
		{"unordered", `data-type="unordered"`, RowTypeUnordered},
		{"task", `data-type="task"`, RowTypeTask},
		{"hr", `data-type="hr"`, RowTypeHR},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := "<p>content</p>"
			if tt.wantType == RowTypeHR {
				p = "<p/>"
			}
			input := bikeDoc(`      <li id="r1" ` + tt.dataType + `>
        ` + p + `
      </li>`)
			doc, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(doc.Rows) != 1 {
				t.Fatalf("Rows = %d, want 1", len(doc.Rows))
			}
			if doc.Rows[0].Type != tt.wantType {
				t.Errorf("Type = %q, want %q", doc.Rows[0].Type, tt.wantType)
			}
		})
	}
}

func TestParseTaskDone(t *testing.T) {
	input := bikeDoc(`      <li id="t1" data-done="2026-02-16T14:30:47Z" data-type="task">
        <p>Completed task</p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if row.Type != RowTypeTask {
		t.Errorf("Type = %q, want %q", row.Type, RowTypeTask)
	}
	if row.DoneAt != "2026-02-16T14:30:47Z" {
		t.Errorf("DoneAt = %q, want %q", row.DoneAt, "2026-02-16T14:30:47Z")
	}
}

func TestParseEmptyRow(t *testing.T) {
	input := bikeDoc(`      <li id="e1">
        <p/>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if row.Content != nil {
		t.Errorf("Content = %v, want nil", row.Content)
	}
}

func TestParseHRRow(t *testing.T) {
	input := bikeDoc(`      <li id="h1" data-type="hr">
        <p/>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if row.Type != RowTypeHR {
		t.Errorf("Type = %q, want %q", row.Type, RowTypeHR)
	}
	if row.Content != nil {
		t.Errorf("Content = %v, want nil", row.Content)
	}
}

func TestParseInlineFormatting(t *testing.T) {
	tests := []struct {
		name  string
		html  string
		check func(t *testing.T, nodes []InlineNode)
	}{
		{
			name: "strong",
			html: `<p><strong>bold</strong></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				if len(nodes) != 1 {
					t.Fatalf("nodes = %d, want 1", len(nodes))
				}
				s, ok := nodes[0].(StrongRun)
				if !ok {
					t.Fatalf("type = %T, want StrongRun", nodes[0])
				}
				assertChildText(t, s.Children, "bold")
			},
		},
		{
			name: "em",
			html: `<p><em>italic</em></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				e, ok := nodes[0].(EmRun)
				if !ok {
					t.Fatalf("type = %T, want EmRun", nodes[0])
				}
				assertChildText(t, e.Children, "italic")
			},
		},
		{
			name: "code",
			html: `<p><code>mono</code></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				c, ok := nodes[0].(CodeRun)
				if !ok {
					t.Fatalf("type = %T, want CodeRun", nodes[0])
				}
				if c.Text != "mono" {
					t.Errorf("Text = %q, want %q", c.Text, "mono")
				}
			},
		},
		{
			name: "link",
			html: `<p><a href="https://example.com">click</a></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				l, ok := nodes[0].(LinkRun)
				if !ok {
					t.Fatalf("type = %T, want LinkRun", nodes[0])
				}
				if l.URL != "https://example.com" {
					t.Errorf("URL = %q, want %q", l.URL, "https://example.com")
				}
				assertChildText(t, l.Children, "click")
			},
		},
		{
			name: "strikethrough",
			html: `<p><s>deleted</s></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				s, ok := nodes[0].(StrikethroughRun)
				if !ok {
					t.Fatalf("type = %T, want StrikethroughRun", nodes[0])
				}
				assertChildText(t, s.Children, "deleted")
			},
		},
		{
			name: "mark",
			html: `<p><mark>highlighted</mark></p>`,
			check: func(t *testing.T, nodes []InlineNode) {
				m, ok := nodes[0].(MarkRun)
				if !ok {
					t.Fatalf("type = %T, want MarkRun", nodes[0])
				}
				assertChildText(t, m.Children, "highlighted")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := bikeDoc(`      <li id="f1">
        ` + tt.html + `
      </li>`)
			doc, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			tt.check(t, doc.Rows[0].Content)
		})
	}
}

func TestParseSpanStripping(t *testing.T) {
	// <span> should be stripped to TextRun
	input := bikeDoc(`      <li id="s1">
        <p><span>Hello </span><strong>world</strong></p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if len(row.Content) != 2 {
		t.Fatalf("Content = %d nodes, want 2", len(row.Content))
	}

	// First node: TextRun from <span>
	tr, ok := row.Content[0].(TextRun)
	if !ok {
		t.Fatalf("Content[0] type = %T, want TextRun", row.Content[0])
	}
	if tr.Text != "Hello " {
		t.Errorf("Text = %q, want %q", tr.Text, "Hello ")
	}

	// Second node: StrongRun
	if _, ok := row.Content[1].(StrongRun); !ok {
		t.Errorf("Content[1] type = %T, want StrongRun", row.Content[1])
	}
}

func TestParseSpanWhitespaceOnly(t *testing.T) {
	// <span> with only whitespace between formatting elements
	input := bikeDoc(`      <li id="w1">
        <p><strong>Size:</strong><span> </span><code>17M</code></p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if len(row.Content) != 3 {
		t.Fatalf("Content = %d nodes, want 3", len(row.Content))
	}

	// Middle node should be TextRun with space
	tr, ok := row.Content[1].(TextRun)
	if !ok {
		t.Fatalf("Content[1] type = %T, want TextRun", row.Content[1])
	}
	if tr.Text != " " {
		t.Errorf("Text = %q, want %q", tr.Text, " ")
	}
}

func TestParseNestedInlines(t *testing.T) {
	input := bikeDoc(`      <li id="n1">
        <p><em><mark>highlighted italic</mark></em></p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := doc.Rows[0]
	if len(row.Content) != 1 {
		t.Fatalf("Content = %d nodes, want 1", len(row.Content))
	}

	em, ok := row.Content[0].(EmRun)
	if !ok {
		t.Fatalf("Content[0] type = %T, want EmRun", row.Content[0])
	}
	if len(em.Children) != 1 {
		t.Fatalf("em.Children = %d, want 1", len(em.Children))
	}

	mark, ok := em.Children[0].(MarkRun)
	if !ok {
		t.Fatalf("em.Children[0] type = %T, want MarkRun", em.Children[0])
	}
	assertChildText(t, mark.Children, "highlighted italic")
}

func TestParseNestedRows(t *testing.T) {
	input := bikeDoc(`      <li id="p1" data-type="heading">
        <p>Parent</p>
        <ul>
          <li id="c1">
            <p>Child one</p>
          </li>
          <li id="c2">
            <p>Child two</p>
          </li>
        </ul>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(doc.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(doc.Rows))
	}
	parent := doc.Rows[0]
	if parent.ID != "p1" {
		t.Errorf("parent.ID = %q, want %q", parent.ID, "p1")
	}
	if len(parent.Children) != 2 {
		t.Fatalf("Children = %d, want 2", len(parent.Children))
	}
	if parent.Children[0].ID != "c1" {
		t.Errorf("child[0].ID = %q, want %q", parent.Children[0].ID, "c1")
	}
	if parent.Children[1].ID != "c2" {
		t.Errorf("child[1].ID = %q, want %q", parent.Children[1].ID, "c2")
	}
}

func TestParseDeeplyNestedRows(t *testing.T) {
	input := bikeDoc(`      <li id="h1" data-type="heading">
        <p>Level 1</p>
        <ul>
          <li id="h2" data-type="heading">
            <p>Level 2</p>
            <ul>
              <li id="b1">
                <p>Deep content</p>
              </li>
            </ul>
          </li>
        </ul>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(doc.Rows) != 1 {
		t.Fatalf("Rows = %d, want 1", len(doc.Rows))
	}
	h1 := doc.Rows[0]
	if len(h1.Children) != 1 {
		t.Fatalf("h1.Children = %d, want 1", len(h1.Children))
	}
	h2 := h1.Children[0]
	if h2.Type != RowTypeHeading {
		t.Errorf("h2.Type = %q, want %q", h2.Type, RowTypeHeading)
	}
	if len(h2.Children) != 1 {
		t.Fatalf("h2.Children = %d, want 1", len(h2.Children))
	}
	if h2.Children[0].ID != "b1" {
		t.Errorf("h2.Children[0].ID = %q, want %q", h2.Children[0].ID, "b1")
	}
}

func TestParseMultipleSiblingRows(t *testing.T) {
	input := bikeDoc(`      <li id="r1">
        <p>First</p>
      </li>
      <li id="r2">
        <p>Second</p>
      </li>
      <li id="r3">
        <p>Third</p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(doc.Rows) != 3 {
		t.Fatalf("Rows = %d, want 3", len(doc.Rows))
	}
	ids := []string{doc.Rows[0].ID, doc.Rows[1].ID, doc.Rows[2].ID}
	want := []string{"r1", "r2", "r3"}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("Rows[%d].ID = %q, want %q", i, id, want[i])
		}
	}
}

func TestParseHTMLEntities(t *testing.T) {
	input := bikeDoc(`      <li id="e1">
        <p>A &amp; B &lt; C</p>
      </li>`)
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tr, ok := doc.Rows[0].Content[0].(TextRun)
	if !ok {
		t.Fatalf("Content[0] type = %T, want TextRun", doc.Rows[0].Content[0])
	}
	if tr.Text != "A & B < C" {
		t.Errorf("Text = %q, want %q", tr.Text, "A & B < C")
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := Parse(strings.NewReader("this is not xml"))
	if err == nil {
		t.Error("Parse() expected error for invalid XML, got nil")
	}
}

func TestParseRoundTrip(t *testing.T) {
	// Build a document, render it, parse it back, and verify structure matches
	original := &Document{
		RootID: "testroot",
		Rows: []*Row{
			{
				ID:   "h1",
				Type: RowTypeHeading,
				Content: []InlineNode{
					TextRun{Text: "Title"},
				},
				Children: []*Row{
					{
						ID:   "b1",
						Type: RowTypeBody,
						Content: []InlineNode{
							TextRun{Text: "Hello "},
							StrongRun{Children: []InlineNode{TextRun{Text: "world"}}},
						},
					},
					{
						ID:      "t1",
						Type:    RowTypeTask,
						DoneAt:  "2026-01-01T00:00:00Z",
						Content: []InlineNode{TextRun{Text: "Done"}},
					},
				},
			},
			{
				ID:   "c1",
				Type: RowTypeCode,
				Content: []InlineNode{
					TextRun{Text: "fmt.Println()"},
				},
			},
			{
				ID:   "hr",
				Type: RowTypeHR,
			},
		},
	}

	// Render to string
	rendered := renderToString(t, original)

	// Parse back
	parsed, err := Parse(strings.NewReader(rendered))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify structure
	if parsed.RootID != original.RootID {
		t.Errorf("RootID = %q, want %q", parsed.RootID, original.RootID)
	}
	if len(parsed.Rows) != len(original.Rows) {
		t.Fatalf("Rows = %d, want %d", len(parsed.Rows), len(original.Rows))
	}

	// Check heading row
	h := parsed.Rows[0]
	if h.Type != RowTypeHeading {
		t.Errorf("Rows[0].Type = %q, want %q", h.Type, RowTypeHeading)
	}
	if h.ID != "h1" {
		t.Errorf("Rows[0].ID = %q, want %q", h.ID, "h1")
	}
	if len(h.Children) != 2 {
		t.Fatalf("Rows[0].Children = %d, want 2", len(h.Children))
	}

	// Check task child
	task := h.Children[1]
	if task.Type != RowTypeTask {
		t.Errorf("task.Type = %q, want %q", task.Type, RowTypeTask)
	}
	if task.DoneAt != "2026-01-01T00:00:00Z" {
		t.Errorf("task.DoneAt = %q, want %q", task.DoneAt, "2026-01-01T00:00:00Z")
	}

	// Check code row
	code := parsed.Rows[1]
	if code.Type != RowTypeCode {
		t.Errorf("code.Type = %q, want %q", code.Type, RowTypeCode)
	}

	// Check HR row
	hr := parsed.Rows[2]
	if hr.Type != RowTypeHR {
		t.Errorf("hr.Type = %q, want %q", hr.Type, RowTypeHR)
	}
	if hr.Content != nil {
		t.Errorf("hr.Content = %v, want nil", hr.Content)
	}
}

// assertChildText checks that a slice of InlineNode contains a single TextRun with the expected text.
func assertChildText(t *testing.T, children []InlineNode, want string) {
	t.Helper()
	if len(children) != 1 {
		t.Fatalf("children = %d, want 1", len(children))
	}
	tr, ok := children[0].(TextRun)
	if !ok {
		t.Fatalf("children[0] type = %T, want TextRun", children[0])
	}
	if tr.Text != want {
		t.Errorf("Text = %q, want %q", tr.Text, want)
	}
}
