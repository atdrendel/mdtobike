package bike

import (
	"fmt"
	"html"
	"io"
	"strings"
)

// Render writes the document as Bike XHTML to the given writer.
func (d *Document) Render(w io.Writer) error {
	bw := &bikeWriter{w: w}
	bw.line(`<?xml version="1.0" encoding="UTF-8"?>`)
	bw.line(`<html>`)
	bw.line(`  <head>`)
	bw.line(`    <meta charset="utf-8"/>`)
	bw.line(`  </head>`)
	bw.line(`  <body>`)
	bw.linef(`    <ul id="%s">`, html.EscapeString(d.RootID))
	for _, row := range d.Rows {
		bw.renderRow(row, 3)
	}
	bw.line(`    </ul>`)
	bw.line(`  </body>`)
	bw.line(`</html>`)
	return bw.err
}

type bikeWriter struct {
	w   io.Writer
	err error
}

func (bw *bikeWriter) write(s string) {
	if bw.err != nil {
		return
	}
	_, bw.err = io.WriteString(bw.w, s)
}

func (bw *bikeWriter) line(s string) {
	bw.write(s)
	bw.write("\n")
}

func (bw *bikeWriter) linef(format string, args ...interface{}) {
	bw.line(fmt.Sprintf(format, args...))
}

func (bw *bikeWriter) indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func (bw *bikeWriter) renderRow(row *Row, depth int) {
	if bw.err != nil {
		return
	}

	ind := bw.indent(depth)

	// Build <li> opening tag
	attrs := fmt.Sprintf(`id="%s"`, html.EscapeString(row.ID))
	if row.DoneAt != "" {
		attrs += fmt.Sprintf(` data-done="%s"`, html.EscapeString(row.DoneAt))
	}
	if row.Type != RowTypeBody {
		attrs += fmt.Sprintf(` data-type="%s"`, string(row.Type))
	}

	if len(row.Children) == 0 {
		// No children: self-contained <li>
		bw.linef(`%s<li %s>`, ind, attrs)
		bw.renderParagraph(row.Content, depth+1)
		bw.linef(`%s</li>`, ind)
	} else {
		// Has children: <li> with <p> then <ul>
		bw.linef(`%s<li %s>`, ind, attrs)
		bw.renderParagraph(row.Content, depth+1)
		bw.linef(`%s<ul>`, bw.indent(depth+1))
		for _, child := range row.Children {
			bw.renderRow(child, depth+2)
		}
		bw.linef(`%s</ul>`, bw.indent(depth+1))
		bw.linef(`%s</li>`, ind)
	}
}

func (bw *bikeWriter) renderParagraph(content []InlineNode, depth int) {
	ind := bw.indent(depth)
	if len(content) == 0 {
		bw.linef(`%s<p/>`, ind)
		return
	}
	bw.write(fmt.Sprintf(`%s<p>`, ind))
	bw.renderInlines(content)
	bw.line(`</p>`)
}

// renderInlines renders inline nodes, applying the <span> wrapping rule.
func (bw *bikeWriter) renderInlines(nodes []InlineNode) {
	needsSpan := hasFormattingNodes(nodes)
	for _, node := range nodes {
		bw.renderInline(node, needsSpan)
	}
}

// hasFormattingNodes returns true if the node list contains any non-TextRun nodes.
func hasFormattingNodes(nodes []InlineNode) bool {
	for _, n := range nodes {
		switch n.(type) {
		case TextRun:
			continue
		default:
			return true
		}
	}
	return false
}

func (bw *bikeWriter) renderInline(node InlineNode, wrapText bool) {
	switch n := node.(type) {
	case TextRun:
		if wrapText {
			bw.write(`<span>`)
			bw.write(html.EscapeString(n.Text))
			bw.write(`</span>`)
		} else {
			bw.write(html.EscapeString(n.Text))
		}
	case StrongRun:
		bw.write(`<strong>`)
		bw.renderInlineChildren(n.Children)
		bw.write(`</strong>`)
	case EmRun:
		bw.write(`<em>`)
		bw.renderInlineChildren(n.Children)
		bw.write(`</em>`)
	case CodeRun:
		bw.write(`<code>`)
		bw.write(html.EscapeString(n.Text))
		bw.write(`</code>`)
	case LinkRun:
		bw.write(fmt.Sprintf(`<a href="%s">`, html.EscapeString(n.URL)))
		bw.renderInlineChildren(n.Children)
		bw.write(`</a>`)
	case StrikethroughRun:
		bw.write(`<s>`)
		bw.renderInlineChildren(n.Children)
		bw.write(`</s>`)
	case MarkRun:
		bw.write(`<mark>`)
		bw.renderInlineChildren(n.Children)
		bw.write(`</mark>`)
	}
}

// renderInlineChildren renders children of a formatting node.
// Inside formatting elements, text is NOT wrapped in <span>.
func (bw *bikeWriter) renderInlineChildren(nodes []InlineNode) {
	for _, node := range nodes {
		bw.renderInline(node, false)
	}
}
