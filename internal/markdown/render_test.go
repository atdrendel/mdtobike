package markdown

import (
	"bytes"
	"strings"
	"testing"

	"github.com/atdrendel/bikemark/internal/bike"
	"github.com/atdrendel/bikemark/internal/convert"
)

func render(t *testing.T, doc *bike.Document) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, doc); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}

func doc(rows ...*bike.Row) *bike.Document {
	return &bike.Document{RootID: "test", Rows: rows}
}

func row(typ bike.RowType, content ...bike.InlineNode) *bike.Row {
	return &bike.Row{ID: "r", Type: typ, Content: content}
}

func textRow(typ bike.RowType, text string) *bike.Row {
	return row(typ, bike.TextRun{Text: text})
}

func TestRenderEmptyDocument(t *testing.T) {
	output := render(t, doc())
	if output != "" {
		t.Errorf("output = %q, want empty", output)
	}
}

func TestRenderBodyRow(t *testing.T) {
	output := render(t, doc(textRow(bike.RowTypeBody, "Hello world")))
	if output != "Hello world\n" {
		t.Errorf("output = %q, want %q", output, "Hello world\n")
	}
}

func TestRenderHeadingDepth0(t *testing.T) {
	output := render(t, doc(textRow(bike.RowTypeHeading, "Title")))
	if output != "# Title\n" {
		t.Errorf("output = %q, want %q", output, "# Title\n")
	}
}

func TestRenderHeadingDepth1(t *testing.T) {
	h1 := textRow(bike.RowTypeHeading, "Top")
	h2 := textRow(bike.RowTypeHeading, "Sub")
	h1.Children = []*bike.Row{h2}
	output := render(t, doc(h1))
	if !strings.Contains(output, "## Sub") {
		t.Errorf("output missing '## Sub':\n%s", output)
	}
}

func TestRenderHeadingDepth2(t *testing.T) {
	h3 := textRow(bike.RowTypeHeading, "Deep")
	h2 := textRow(bike.RowTypeHeading, "Mid")
	h2.Children = []*bike.Row{h3}
	h1 := textRow(bike.RowTypeHeading, "Top")
	h1.Children = []*bike.Row{h2}
	output := render(t, doc(h1))
	if !strings.Contains(output, "### Deep") {
		t.Errorf("output missing '### Deep':\n%s", output)
	}
}

func TestRenderHeadingWithBodyChildren(t *testing.T) {
	h := textRow(bike.RowTypeHeading, "Title")
	h.Children = []*bike.Row{textRow(bike.RowTypeBody, "Paragraph")}
	output := render(t, doc(h))
	if !strings.Contains(output, "# Title") {
		t.Errorf("output missing '# Title':\n%s", output)
	}
	if !strings.Contains(output, "Paragraph") {
		t.Errorf("output missing 'Paragraph':\n%s", output)
	}
}

func TestRenderHeadingMaxLevel(t *testing.T) {
	// Build 7 levels deep — should cap at ######
	current := textRow(bike.RowTypeHeading, "L7")
	for i := 6; i >= 1; i-- {
		parent := textRow(bike.RowTypeHeading, "L"+string(rune('0'+i)))
		parent.Children = []*bike.Row{current}
		current = parent
	}
	output := render(t, doc(current))
	// Level 7 should still be ###### (capped at 6)
	if strings.Contains(output, "####### ") {
		t.Errorf("output contains 7 hashes:\n%s", output)
	}
	if !strings.Contains(output, "###### L7") {
		t.Errorf("output missing '###### L7':\n%s", output)
	}
}

func TestRenderQuoteRow(t *testing.T) {
	output := render(t, doc(textRow(bike.RowTypeQuote, "Quoted text")))
	if !strings.Contains(output, "> Quoted text") {
		t.Errorf("output missing '> Quoted text':\n%s", output)
	}
}

func TestRenderConsecutiveQuoteRows(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeQuote, "Line one"),
		textRow(bike.RowTypeQuote, "Line two"),
	))
	if !strings.Contains(output, "> Line one\n> Line two") {
		t.Errorf("output missing consecutive quote lines:\n%s", output)
	}
}

func TestRenderSingleCodeRow(t *testing.T) {
	output := render(t, doc(textRow(bike.RowTypeCode, "hello()")))
	want := "```\nhello()\n```\n"
	if output != want {
		t.Errorf("output = %q, want %q", output, want)
	}
}

func TestRenderConsecutiveCodeRows(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeCode, "line 1"),
		textRow(bike.RowTypeCode, "line 2"),
		textRow(bike.RowTypeCode, "line 3"),
	))
	want := "```\nline 1\nline 2\nline 3\n```\n"
	if output != want {
		t.Errorf("output = %q, want %q", output, want)
	}
}

func TestRenderCodeRowEmptyContent(t *testing.T) {
	empty := &bike.Row{ID: "r", Type: bike.RowTypeCode, Content: nil}
	output := render(t, doc(
		textRow(bike.RowTypeCode, "before"),
		empty,
		textRow(bike.RowTypeCode, "after"),
	))
	want := "```\nbefore\n\nafter\n```\n"
	if output != want {
		t.Errorf("output = %q, want %q", output, want)
	}
}

func TestRenderHRRow(t *testing.T) {
	output := render(t, doc(&bike.Row{ID: "r", Type: bike.RowTypeHR}))
	if !strings.Contains(output, "---") {
		t.Errorf("output missing '---':\n%s", output)
	}
}

func TestRenderUnorderedList(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeUnordered, "Alpha"),
		textRow(bike.RowTypeUnordered, "Beta"),
	))
	if !strings.Contains(output, "- Alpha\n- Beta") {
		t.Errorf("output missing unordered list:\n%s", output)
	}
}

func TestRenderOrderedList(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeOrdered, "First"),
		textRow(bike.RowTypeOrdered, "Second"),
	))
	if !strings.Contains(output, "1. First\n1. Second") {
		t.Errorf("output missing ordered list:\n%s", output)
	}
}

func TestRenderTaskUnchecked(t *testing.T) {
	output := render(t, doc(textRow(bike.RowTypeTask, "Todo")))
	if !strings.Contains(output, "- [ ] Todo") {
		t.Errorf("output missing '- [ ] Todo':\n%s", output)
	}
}

func TestRenderTaskChecked(t *testing.T) {
	r := textRow(bike.RowTypeTask, "Done")
	r.DoneAt = "2026-01-01T00:00:00Z"
	output := render(t, doc(r))
	if !strings.Contains(output, "- [x] Done") {
		t.Errorf("output missing '- [x] Done':\n%s", output)
	}
}

func TestRenderNestedUnorderedList(t *testing.T) {
	parent := textRow(bike.RowTypeUnordered, "Parent")
	parent.Children = []*bike.Row{
		textRow(bike.RowTypeUnordered, "Child"),
	}
	output := render(t, doc(parent))
	if !strings.Contains(output, "- Parent\n  - Child") {
		t.Errorf("output missing nested list:\n%s", output)
	}
}

func TestRenderNestedOrderedList(t *testing.T) {
	parent := textRow(bike.RowTypeOrdered, "Parent")
	parent.Children = []*bike.Row{
		textRow(bike.RowTypeOrdered, "Child"),
	}
	output := render(t, doc(parent))
	if !strings.Contains(output, "1. Parent\n   1. Child") {
		t.Errorf("output missing nested ordered list:\n%s", output)
	}
}

func TestRenderNoteRow(t *testing.T) {
	// Note rows render as paragraphs (lossy)
	note := &bike.Row{
		ID:      "r",
		Type:    bike.RowType("note"),
		Content: []bike.InlineNode{bike.TextRun{Text: "A note"}},
	}
	output := render(t, doc(note))
	if !strings.Contains(output, "A note") {
		t.Errorf("output missing note text:\n%s", output)
	}
}

func TestRenderInlineStrong(t *testing.T) {
	r := row(bike.RowTypeBody, bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "bold"}}})
	output := render(t, doc(r))
	if !strings.Contains(output, "**bold**") {
		t.Errorf("output missing '**bold**':\n%s", output)
	}
}

func TestRenderInlineEm(t *testing.T) {
	r := row(bike.RowTypeBody, bike.EmRun{Children: []bike.InlineNode{bike.TextRun{Text: "italic"}}})
	output := render(t, doc(r))
	if !strings.Contains(output, "*italic*") {
		t.Errorf("output missing '*italic*':\n%s", output)
	}
}

func TestRenderInlineCode(t *testing.T) {
	r := row(bike.RowTypeBody, bike.CodeRun{Text: "code"})
	output := render(t, doc(r))
	if !strings.Contains(output, "`code`") {
		t.Errorf("output missing '`code`':\n%s", output)
	}
}

func TestRenderInlineLink(t *testing.T) {
	r := row(bike.RowTypeBody, bike.LinkRun{
		URL:      "https://example.com",
		Children: []bike.InlineNode{bike.TextRun{Text: "click"}},
	})
	output := render(t, doc(r))
	if !strings.Contains(output, "[click](https://example.com)") {
		t.Errorf("output missing link:\n%s", output)
	}
}

func TestRenderInlineStrikethrough(t *testing.T) {
	r := row(bike.RowTypeBody, bike.StrikethroughRun{Children: []bike.InlineNode{bike.TextRun{Text: "gone"}}})
	output := render(t, doc(r))
	if !strings.Contains(output, "~~gone~~") {
		t.Errorf("output missing '~~gone~~':\n%s", output)
	}
}

func TestRenderInlineMark(t *testing.T) {
	r := row(bike.RowTypeBody, bike.MarkRun{Children: []bike.InlineNode{bike.TextRun{Text: "hilite"}}})
	output := render(t, doc(r))
	if !strings.Contains(output, "==hilite==") {
		t.Errorf("output missing '==hilite==':\n%s", output)
	}
}

func TestRenderNestedInlines(t *testing.T) {
	r := row(bike.RowTypeBody, bike.EmRun{
		Children: []bike.InlineNode{
			bike.MarkRun{Children: []bike.InlineNode{bike.TextRun{Text: "text"}}},
		},
	})
	output := render(t, doc(r))
	if !strings.Contains(output, "*==text==*") {
		t.Errorf("output missing '*==text==*':\n%s", output)
	}
}

func TestRenderMixedInlines(t *testing.T) {
	r := row(bike.RowTypeBody,
		bike.TextRun{Text: "Hello "},
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "world"}}},
		bike.TextRun{Text: "!"},
	)
	output := render(t, doc(r))
	if !strings.Contains(output, "Hello **world**!") {
		t.Errorf("output missing 'Hello **world**!':\n%s", output)
	}
}

func TestRenderBlankLineSeparation(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeBody, "Para one"),
		textRow(bike.RowTypeBody, "Para two"),
	))
	if !strings.Contains(output, "Para one\n\nPara two") {
		t.Errorf("output missing blank line between paragraphs:\n%s", output)
	}
}

func TestRenderNoBlankLineBetweenListItems(t *testing.T) {
	output := render(t, doc(
		textRow(bike.RowTypeUnordered, "One"),
		textRow(bike.RowTypeUnordered, "Two"),
	))
	if strings.Contains(output, "- One\n\n- Two") {
		t.Errorf("output has unwanted blank line between list items:\n%s", output)
	}
}

func TestRenderListItemWithChildren(t *testing.T) {
	// Bug: when a list follows a heading (which sets needsBlankLine=true),
	// the stale needsBlankLine causes a spurious blank line between a list
	// item and its nested children.
	heading := textRow(bike.RowTypeHeading, "Section")
	task := textRow(bike.RowTypeTask, "Parent task")
	task.Children = []*bike.Row{
		textRow(bike.RowTypeUnordered, "Child link"),
	}
	heading.Children = []*bike.Row{task}
	output := render(t, doc(heading))
	// Must NOT have a blank line between parent and child
	if !strings.Contains(output, "- [ ] Parent task\n  - Child link\n") {
		t.Errorf("output has wrong nesting:\n%s", output)
	}
}

func TestRenderListItemWithDeeplyNestedChildren(t *testing.T) {
	heading := textRow(bike.RowTypeHeading, "Section")
	child := textRow(bike.RowTypeUnordered, "Grandchild")
	ordered := textRow(bike.RowTypeOrdered, "Step 1")
	ordered.Children = []*bike.Row{child}
	task := textRow(bike.RowTypeTask, "Parent")
	task.Children = []*bike.Row{ordered}
	heading.Children = []*bike.Row{task}
	output := render(t, doc(heading))
	// Only one blank line allowed (after the heading), not within the list nesting
	parts := strings.SplitN(output, "\n\n", 3)
	if len(parts) > 2 {
		t.Errorf("output has more than one blank line:\n%s", output)
	}
}

func TestRenderTaskChildRoundTrip(t *testing.T) {
	// Regression: task item children must survive a Markdown round-trip.
	// If the child indent is too wide (e.g. 6 spaces for "- [ ] "), goldmark
	// treats the child as continuation text instead of a sublist, merging it
	// into the parent's content.
	task := textRow(bike.RowTypeTask, "Parent")
	task.Children = []*bike.Row{
		textRow(bike.RowTypeUnordered, "Child"),
	}
	heading := textRow(bike.RowTypeHeading, "Section")
	heading.Children = []*bike.Row{task}

	md := render(t, doc(heading))

	// Re-parse the Markdown through the full pipeline
	bikeDoc, err := convert.FromMarkdown([]byte(md))
	if err != nil {
		t.Fatalf("FromMarkdown() error = %v", err)
	}

	// Navigate to the task row and check it has a child
	if len(bikeDoc.Rows) < 1 {
		t.Fatalf("expected at least 1 top-level row, got %d", len(bikeDoc.Rows))
	}
	section := bikeDoc.Rows[0]
	if len(section.Children) < 1 {
		t.Fatalf("expected heading to have children, got %d", len(section.Children))
	}
	taskRow := section.Children[0]
	if taskRow.Type != bike.RowTypeTask {
		t.Fatalf("expected task row, got %q", taskRow.Type)
	}
	if len(taskRow.Children) != 1 {
		t.Errorf("task should have 1 child, got %d; markdown was:\n%s", len(taskRow.Children), md)
	}
}

func TestRenderAdjacentStrongRunsWithEm(t *testing.T) {
	// Bike stores "**text(**_**word**_**)**" as three sibling StrongRuns.
	// Without normalization, this produces ambiguous Markdown that goldmark
	// parses into a different (duplicate-nested) structure.
	// After normalization, should produce: **text(*word*)**
	r := row(bike.RowTypeBody,
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "Permit ("}}},
		bike.EmRun{Children: []bike.InlineNode{
			bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "Aufenthaltstitel"}}},
		}},
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: ")"}}},
	)
	output := render(t, doc(r))
	if !strings.Contains(output, "**Permit (*Aufenthaltstitel*)**") {
		t.Errorf("expected merged strong with nested em, got:\n%s", output)
	}
}

func TestRenderAdjacentStrongRunsSimple(t *testing.T) {
	// Two adjacent StrongRuns should merge into one
	r := row(bike.RowTypeBody,
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "hello "}}},
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "world"}}},
	)
	output := render(t, doc(r))
	if !strings.Contains(output, "**hello world**") {
		t.Errorf("expected merged strong, got:\n%s", output)
	}
}

func TestRenderAdjacentStrongRoundTrip(t *testing.T) {
	// The key test: adjacent strong runs with em between them must survive
	// a Markdown round-trip without producing duplicate <strong> nesting.
	r := row(bike.RowTypeBody,
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "Permit ("}}},
		bike.EmRun{Children: []bike.InlineNode{
			bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: "Aufenthaltstitel"}}},
		}},
		bike.StrongRun{Children: []bike.InlineNode{bike.TextRun{Text: ")"}}},
	)
	md := render(t, doc(r))

	// Re-parse and re-render — should NOT have <strong><strong>
	bikeDoc, err := convert.FromMarkdown([]byte(md))
	if err != nil {
		t.Fatalf("FromMarkdown() error = %v", err)
	}
	var buf bytes.Buffer
	if err := bikeDoc.Render(&buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	bikeOutput := buf.String()
	if strings.Contains(bikeOutput, "<strong><strong>") {
		t.Errorf("round-trip produced duplicate <strong> nesting; bike output:\n%s\nmarkdown was:\n%s", bikeOutput, md)
	}
}

func TestRenderComplexDocument(t *testing.T) {
	heading := textRow(bike.RowTypeHeading, "My Doc")
	heading.Children = []*bike.Row{
		textRow(bike.RowTypeBody, "Intro paragraph"),
		textRow(bike.RowTypeCode, "line 1"),
		textRow(bike.RowTypeCode, "line 2"),
		textRow(bike.RowTypeQuote, "A quote"),
		{ID: "hr", Type: bike.RowTypeHR},
		textRow(bike.RowTypeUnordered, "Item A"),
		textRow(bike.RowTypeUnordered, "Item B"),
	}

	checkedTask := textRow(bike.RowTypeTask, "Done")
	checkedTask.DoneAt = "2026-01-01T00:00:00Z"

	output := render(t, doc(heading, checkedTask))

	checks := []string{
		"# My Doc",
		"Intro paragraph",
		"```\nline 1\nline 2\n```",
		"> A quote",
		"---",
		"- Item A\n- Item B",
		"- [x] Done",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}
