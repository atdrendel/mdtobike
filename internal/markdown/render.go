// Package markdown renders a Bike document model as GitHub-flavored Markdown text.
package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/atdrendel/bikemark/internal/bike"
)

// Render writes the Document as GFM Markdown to w.
func Render(w io.Writer, doc *bike.Document) error {
	mw := &markdownWriter{w: w}
	mw.renderRows(doc.Rows, 0, "")
	return mw.err
}

// markdownWriter accumulates output and tracks the first error.
type markdownWriter struct {
	w              io.Writer
	err            error
	needsBlankLine bool
}

func (mw *markdownWriter) write(s string) {
	if mw.err != nil {
		return
	}
	_, mw.err = io.WriteString(mw.w, s)
}

// blankLine emits a blank line separator if needed.
func (mw *markdownWriter) blankLine() {
	if mw.needsBlankLine {
		mw.write("\n")
	}
}

// renderRows renders a slice of rows at the given heading depth and indent prefix.
// It groups consecutive "tight" rows (code, quote, list) so blank lines are managed
// at the group boundary rather than between individual rows.
func (mw *markdownWriter) renderRows(rows []*bike.Row, headingDepth int, indent string) {
	i := 0
	for i < len(rows) {
		row := rows[i]

		// Group consecutive code rows into a single fenced block
		if row.Type == bike.RowTypeCode {
			j := i + 1
			for j < len(rows) && rows[j].Type == bike.RowTypeCode {
				j++
			}
			mw.renderCodeBlock(rows[i:j], indent)
			i = j
			continue
		}

		// Group consecutive quote rows
		if row.Type == bike.RowTypeQuote {
			j := i + 1
			for j < len(rows) && rows[j].Type == bike.RowTypeQuote {
				j++
			}
			mw.renderQuoteBlock(rows[i:j], indent)
			i = j
			continue
		}

		// Group consecutive list items (unordered, ordered, task)
		if isListType(row.Type) {
			j := i + 1
			for j < len(rows) && isListType(rows[j].Type) {
				j++
			}
			mw.renderListBlock(rows[i:j], headingDepth, indent)
			i = j
			continue
		}

		mw.renderRow(row, headingDepth, indent)
		i++
	}
}

func isListType(t bike.RowType) bool {
	return t == bike.RowTypeUnordered || t == bike.RowTypeOrdered || t == bike.RowTypeTask
}

// renderRow renders a single non-grouped row.
func (mw *markdownWriter) renderRow(row *bike.Row, headingDepth int, indent string) {
	switch row.Type {
	case bike.RowTypeHeading:
		mw.blankLine()
		level := headingDepth + 1
		if level > 6 {
			level = 6
		}
		mw.write(indent + strings.Repeat("#", level) + " " + renderInlines(row.Content) + "\n")
		mw.needsBlankLine = true
		mw.renderRows(row.Children, headingDepth+1, indent)

	case bike.RowTypeHR:
		mw.blankLine()
		mw.write(indent + "---\n")
		mw.needsBlankLine = true

	default:
		// Body, note, or unknown — render as paragraph
		mw.blankLine()
		mw.write(indent + renderInlines(row.Content) + "\n")
		mw.needsBlankLine = true
		mw.renderRows(row.Children, headingDepth, indent)
	}
}

// renderQuoteBlock renders a group of consecutive quote rows.
func (mw *markdownWriter) renderQuoteBlock(rows []*bike.Row, indent string) {
	mw.blankLine()
	for _, row := range rows {
		content := renderInlines(row.Content)
		if content == "" {
			mw.write(indent + ">\n")
		} else {
			mw.write(indent + "> " + content + "\n")
		}
	}
	mw.needsBlankLine = true
}

// renderListBlock renders a group of consecutive list items tightly (no blank lines between).
func (mw *markdownWriter) renderListBlock(rows []*bike.Row, headingDepth int, indent string) {
	mw.blankLine()
	for _, row := range rows {
		prefix := listPrefix(row)
		mw.write(indent + prefix + renderInlines(row.Content) + "\n")
		mw.needsBlankLine = false // no blank line between item and its children
		if len(row.Children) > 0 {
			childIndent := indent + strings.Repeat(" ", listChildIndent(row))
			mw.renderRows(row.Children, headingDepth, childIndent)
		}
	}
	mw.needsBlankLine = true
}

// listChildIndent returns the CommonMark-compatible indentation width for
// children of a list item. This is the marker width, not the full prefix width
// (which includes the checkbox for task items). CommonMark only allows 0-3
// spaces of indent for a nested list marker relative to the parent content
// start; using the full prefix width (6 for tasks) exceeds this threshold.
func listChildIndent(row *bike.Row) int {
	if row.Type == bike.RowTypeOrdered {
		return 3
	}
	return 2
}

func listPrefix(row *bike.Row) string {
	switch row.Type {
	case bike.RowTypeOrdered:
		return "1. "
	case bike.RowTypeTask:
		if row.DoneAt != "" {
			return "- [x] "
		}
		return "- [ ] "
	default:
		return "- "
	}
}

// renderCodeBlock renders consecutive code rows as a single fenced code block.
func (mw *markdownWriter) renderCodeBlock(rows []*bike.Row, indent string) {
	mw.blankLine()
	mw.write(indent + "```\n")
	for _, row := range rows {
		mw.write(indent + renderInlines(row.Content) + "\n")
	}
	mw.write(indent + "```\n")
	mw.needsBlankLine = true
}

// renderInlines converts a slice of InlineNode to Markdown text.
func renderInlines(nodes []bike.InlineNode) string {
	nodes = normalizeInlines(nodes)
	var sb strings.Builder
	for _, node := range nodes {
		renderInline(&sb, node)
	}
	return sb.String()
}

// normalizeInlines merges adjacent same-type formatting runs to produce
// unambiguous Markdown. For example:
//
//	[Strong("a"), Em(Strong("b")), Strong("c")]
//
// becomes:
//
//	[Strong("a", Em("b"), "c")]
//
// This prevents ambiguous asterisk sequences that goldmark would parse
// into different nesting structures.
func normalizeInlines(nodes []bike.InlineNode) []bike.InlineNode {
	if len(nodes) <= 1 {
		return nodes
	}
	result := make([]bike.InlineNode, 0, len(nodes))
	for i := 0; i < len(nodes); i++ {
		switch v := nodes[i].(type) {
		case bike.StrongRun:
			merged := make([]bike.InlineNode, len(v.Children))
			copy(merged, v.Children)
			for i+1 < len(nodes) {
				next := nodes[i+1]
				if ns, ok := next.(bike.StrongRun); ok {
					// Adjacent StrongRun — merge children
					merged = append(merged, ns.Children...)
					i++
				} else if canAbsorbIntoStrong(next) {
					// e.g., EmRun whose sole child is StrongRun — absorb
					merged = append(merged, unwrapInnerStrong(next))
					i++
				} else {
					break
				}
			}
			result = append(result, bike.StrongRun{Children: normalizeInlines(merged)})
		case bike.EmRun:
			merged := make([]bike.InlineNode, len(v.Children))
			copy(merged, v.Children)
			for i+1 < len(nodes) {
				next := nodes[i+1]
				if ne, ok := next.(bike.EmRun); ok {
					merged = append(merged, ne.Children...)
					i++
				} else if canAbsorbIntoEm(next) {
					merged = append(merged, unwrapInnerEm(next))
					i++
				} else {
					break
				}
			}
			result = append(result, bike.EmRun{Children: normalizeInlines(merged)})
		default:
			result = append(result, nodes[i])
		}
	}
	return result
}

// canAbsorbIntoStrong returns true if the node is an EmRun whose sole child
// is a StrongRun. This pattern (Em(Strong("x"))) can be absorbed into an
// adjacent StrongRun as Em("x"), eliminating the redundant inner Strong.
func canAbsorbIntoStrong(node bike.InlineNode) bool {
	em, ok := node.(bike.EmRun)
	if !ok || len(em.Children) != 1 {
		return false
	}
	_, ok = em.Children[0].(bike.StrongRun)
	return ok
}

// unwrapInnerStrong transforms Em(Strong("x")) into Em("x") for absorption
// into a parent StrongRun.
func unwrapInnerStrong(node bike.InlineNode) bike.InlineNode {
	em := node.(bike.EmRun)
	inner := em.Children[0].(bike.StrongRun)
	return bike.EmRun{Children: inner.Children}
}

// canAbsorbIntoEm returns true if the node is a StrongRun whose sole child
// is an EmRun.
func canAbsorbIntoEm(node bike.InlineNode) bool {
	s, ok := node.(bike.StrongRun)
	if !ok || len(s.Children) != 1 {
		return false
	}
	_, ok = s.Children[0].(bike.EmRun)
	return ok
}

// unwrapInnerEm transforms Strong(Em("x")) into Strong("x") for absorption
// into a parent EmRun.
func unwrapInnerEm(node bike.InlineNode) bike.InlineNode {
	s := node.(bike.StrongRun)
	inner := s.Children[0].(bike.EmRun)
	return bike.StrongRun{Children: inner.Children}
}

// renderInline renders a single inline node to the string builder.
func renderInline(sb *strings.Builder, node bike.InlineNode) {
	switch v := node.(type) {
	case bike.TextRun:
		sb.WriteString(v.Text)
	case bike.StrongRun:
		sb.WriteString("**")
		renderInlineChildren(sb, v.Children)
		sb.WriteString("**")
	case bike.EmRun:
		sb.WriteString("*")
		renderInlineChildren(sb, v.Children)
		sb.WriteString("*")
	case bike.CodeRun:
		sb.WriteString("`")
		sb.WriteString(v.Text)
		sb.WriteString("`")
	case bike.LinkRun:
		sb.WriteString("[")
		renderInlineChildren(sb, v.Children)
		sb.WriteString(fmt.Sprintf("](%s)", v.URL))
	case bike.StrikethroughRun:
		sb.WriteString("~~")
		renderInlineChildren(sb, v.Children)
		sb.WriteString("~~")
	case bike.MarkRun:
		sb.WriteString("==")
		renderInlineChildren(sb, v.Children)
		sb.WriteString("==")
	}
}

func renderInlineChildren(sb *strings.Builder, children []bike.InlineNode) {
	for _, child := range children {
		renderInline(sb, child)
	}
}
