// Package convert transforms GitHub-flavored Markdown into a Bike document model.
package convert

import (
	"fmt"
	"strings"
	"time"

	"github.com/atdrendel/bikemark/internal/bike"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	highlight "github.com/zuern/goldmark-highlight"
)

// FromMarkdown parses Markdown source and returns a Bike document.
func FromMarkdown(source []byte) (*bike.Document, error) {
	return fromMarkdown(source, time.Now)
}

// fromMarkdown is the testable version that accepts a clock function.
func fromMarkdown(source []byte, now func() time.Time) (*bike.Document, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&highlight.Extender{},
		),
	)

	doc := md.Parser().Parse(text.NewReader(source))
	idGen := &bike.IDGenerator{}

	bikeDoc := &bike.Document{
		RootID: idGen.RootID(),
	}

	c := &converter{
		source: source,
		idGen:  idGen,
		now:    now,
	}

	c.convertChildren(doc, bikeDoc, idGen)
	return bikeDoc, nil
}

type converter struct {
	source []byte
	idGen  *bike.IDGenerator
	now    func() time.Time
}

// convertChildren processes top-level block nodes and builds the heading hierarchy.
func (c *converter) convertChildren(doc ast.Node, bikeDoc *bike.Document, idGen *bike.IDGenerator) {
	var stack []stackEntry // heading hierarchy stack

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		rows := c.convertBlock(child)
		for _, row := range rows {
			if row.Type == bike.RowTypeHeading {
				level := c.headingLevel(child)
				// Pop entries with level >= this heading's level
				for len(stack) > 0 && stack[len(stack)-1].level >= level {
					stack = stack[:len(stack)-1]
				}
				if len(stack) == 0 {
					bikeDoc.Rows = append(bikeDoc.Rows, row)
				} else {
					parent := stack[len(stack)-1].row
					parent.Children = append(parent.Children, row)
				}
				stack = append(stack, stackEntry{level: level, row: row})
			} else {
				if len(stack) == 0 {
					bikeDoc.Rows = append(bikeDoc.Rows, row)
				} else {
					parent := stack[len(stack)-1].row
					parent.Children = append(parent.Children, row)
				}
			}
		}
	}
}

type stackEntry struct {
	level int
	row   *bike.Row
}

// headingLevel returns the heading level for a node, or 0 if not a heading.
func (c *converter) headingLevel(n ast.Node) int {
	if h, ok := n.(*ast.Heading); ok {
		return h.Level
	}
	return 0
}

// convertBlock converts a single top-level AST block node into one or more Bike rows.
func (c *converter) convertBlock(n ast.Node) []*bike.Row {
	switch node := n.(type) {
	case *ast.Heading:
		return []*bike.Row{c.convertHeading(node)}
	case *ast.Paragraph:
		return []*bike.Row{c.convertParagraph(node)}
	case *ast.FencedCodeBlock:
		return c.convertFencedCodeBlock(node)
	case *ast.CodeBlock:
		return c.convertCodeBlock(node)
	case *ast.Blockquote:
		return c.convertBlockquote(node)
	case *ast.List:
		return c.convertList(node)
	case *extast.Table:
		return c.convertTable(node)
	case *ast.ThematicBreak:
		return []*bike.Row{c.convertThematicBreak()}
	case *ast.HTMLBlock:
		return []*bike.Row{c.convertHTMLBlock(node)}
	default:
		// Unknown block type — render as empty body row
		return []*bike.Row{{
			ID:   c.idGen.Next(),
			Type: bike.RowTypeBody,
		}}
	}
}

func (c *converter) convertHeading(n *ast.Heading) *bike.Row {
	return &bike.Row{
		ID:      c.idGen.Next(),
		Type:    bike.RowTypeHeading,
		Content: c.extractInlines(n),
	}
}

func (c *converter) convertParagraph(n *ast.Paragraph) *bike.Row {
	return &bike.Row{
		ID:      c.idGen.Next(),
		Type:    bike.RowTypeBody,
		Content: c.extractInlines(n),
	}
}

func (c *converter) convertFencedCodeBlock(n *ast.FencedCodeBlock) []*bike.Row {
	var rows []*bike.Row
	// Collect all lines from the code block
	var content strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		content.Write(line.Value(c.source))
	}
	// Split into lines and create one code row per line
	text := strings.TrimRight(content.String(), "\n")
	if text == "" {
		return []*bike.Row{{
			ID:   c.idGen.Next(),
			Type: bike.RowTypeCode,
		}}
	}
	for _, line := range strings.Split(text, "\n") {
		rows = append(rows, &bike.Row{
			ID:      c.idGen.Next(),
			Type:    bike.RowTypeCode,
			Content: []bike.InlineNode{bike.TextRun{Text: line}},
		})
	}
	return rows
}

func (c *converter) convertCodeBlock(n *ast.CodeBlock) []*bike.Row {
	var rows []*bike.Row
	var content strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		content.Write(line.Value(c.source))
	}
	text := strings.TrimRight(content.String(), "\n")
	if text == "" {
		return []*bike.Row{{
			ID:   c.idGen.Next(),
			Type: bike.RowTypeCode,
		}}
	}
	for _, line := range strings.Split(text, "\n") {
		rows = append(rows, &bike.Row{
			ID:      c.idGen.Next(),
			Type:    bike.RowTypeCode,
			Content: []bike.InlineNode{bike.TextRun{Text: line}},
		})
	}
	return rows
}

func (c *converter) convertBlockquote(n *ast.Blockquote) []*bike.Row {
	var rows []*bike.Row
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if para, ok := child.(*ast.Paragraph); ok {
			rows = append(rows, &bike.Row{
				ID:      c.idGen.Next(),
				Type:    bike.RowTypeQuote,
				Content: c.extractInlines(para),
			})
		} else {
			// Nested blockquote or other block inside blockquote
			inner := c.convertBlock(child)
			for _, r := range inner {
				r.Type = bike.RowTypeQuote
				rows = append(rows, r)
			}
		}
	}
	return rows
}

func (c *converter) convertList(n *ast.List) []*bike.Row {
	var rows []*bike.Row
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if item, ok := child.(*ast.ListItem); ok {
			rows = append(rows, c.convertListItem(item, n))
		}
	}
	return rows
}

func (c *converter) convertListItem(item *ast.ListItem, list *ast.List) *bike.Row {
	row := &bike.Row{
		ID: c.idGen.Next(),
	}

	// Determine row type — check for task checkbox first
	isTask := false
	var taskChecked bool
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		// Task checkboxes can be inside Paragraph (loose lists) or TextBlock (tight lists)
		if fc := child.FirstChild(); fc != nil {
			if cb, ok := fc.(*extast.TaskCheckBox); ok {
				isTask = true
				taskChecked = cb.IsChecked
				break
			}
		}
	}

	if isTask {
		row.Type = bike.RowTypeTask
		if taskChecked {
			row.DoneAt = c.now().UTC().Format(time.RFC3339)
		}
	} else if list.IsOrdered() {
		row.Type = bike.RowTypeOrdered
	} else {
		row.Type = bike.RowTypeUnordered
	}

	// Extract content and children
	// goldmark uses *ast.Paragraph for loose lists and *ast.TextBlock for tight lists
	for child := item.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.(type) {
		case *ast.Paragraph, *ast.TextBlock:
			row.Content = c.extractInlines(child)
		case *ast.List:
			// Nested list — becomes children of this row
			row.Children = append(row.Children, c.convertList(child.(*ast.List))...)
		default:
			// Other block types inside list items
			inner := c.convertBlock(child)
			row.Children = append(row.Children, inner...)
		}
	}

	return row
}

func (c *converter) convertThematicBreak() *bike.Row {
	return &bike.Row{
		ID:   c.idGen.Next(),
		Type: bike.RowTypeHR,
	}
}

func (c *converter) convertHTMLBlock(n *ast.HTMLBlock) *bike.Row {
	var content strings.Builder
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		content.Write(line.Value(c.source))
	}
	text := strings.TrimSpace(content.String())
	return &bike.Row{
		ID:      c.idGen.Next(),
		Type:    bike.RowTypeBody,
		Content: []bike.InlineNode{bike.TextRun{Text: text}},
	}
}

func (c *converter) convertTable(n *extast.Table) []*bike.Row {
	// Header row becomes a parent body row; data rows become its children.
	// Cells within each row are joined with " — ".
	var headerRow *bike.Row
	var dataRows []*bike.Row

	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		content := c.extractTableRowCells(child)
		row := &bike.Row{
			ID:      c.idGen.Next(),
			Type:    bike.RowTypeBody,
			Content: content,
		}
		switch child.(type) {
		case *extast.TableHeader:
			headerRow = row
		case *extast.TableRow:
			dataRows = append(dataRows, row)
		}
	}

	if headerRow != nil {
		headerRow.Children = dataRows
		return []*bike.Row{headerRow}
	}
	// No header — return data rows as siblings
	return dataRows
}

// extractTableRowCells extracts inline content from all cells in a table row,
// joining them with " — " separators.
func (c *converter) extractTableRowCells(row ast.Node) []bike.InlineNode {
	var result []bike.InlineNode
	first := true
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if !first {
			result = append(result, bike.TextRun{Text: " — "})
		}
		first = false
		result = append(result, c.extractInlines(cell)...)
	}
	return mergeTextRuns(result)
}

// extractInlines collects inline content from a block node's children.
func (c *converter) extractInlines(n ast.Node) []bike.InlineNode {
	var result []bike.InlineNode
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		result = append(result, c.convertInline(child)...)
	}
	return mergeTextRuns(result)
}

// mergeTextRuns coalesces adjacent TextRun nodes in a slice of InlineNodes.
// This prevents goldmark's text-node splitting from producing multiple <span>
// elements where the original had one.
func mergeTextRuns(nodes []bike.InlineNode) []bike.InlineNode {
	if len(nodes) <= 1 {
		return nodes
	}
	result := make([]bike.InlineNode, 0, len(nodes))
	for _, node := range nodes {
		if tr, ok := node.(bike.TextRun); ok {
			if len(result) > 0 {
				if prev, ok := result[len(result)-1].(bike.TextRun); ok {
					result[len(result)-1] = bike.TextRun{Text: prev.Text + tr.Text}
					continue
				}
			}
		}
		result = append(result, node)
	}
	return result
}

// convertInline converts an inline AST node to bike InlineNodes.
func (c *converter) convertInline(n ast.Node) []bike.InlineNode {
	switch node := n.(type) {
	case *ast.Text:
		text := string(node.Segment.Value(c.source))
		if node.SoftLineBreak() {
			text += " "
		}
		return []bike.InlineNode{bike.TextRun{Text: text}}
	case *ast.String:
		return []bike.InlineNode{bike.TextRun{Text: string(node.Value)}}
	case *ast.Emphasis:
		children := c.extractInlines(node)
		if node.Level == 2 {
			return []bike.InlineNode{bike.StrongRun{Children: children}}
		}
		return []bike.InlineNode{bike.EmRun{Children: children}}
	case *ast.CodeSpan:
		var text strings.Builder
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if t, ok := child.(*ast.Text); ok {
				text.Write(t.Segment.Value(c.source))
			} else if s, ok := child.(*ast.String); ok {
				text.Write(s.Value)
			}
		}
		return []bike.InlineNode{bike.CodeRun{Text: text.String()}}
	case *ast.Link:
		children := c.extractInlines(node)
		return []bike.InlineNode{bike.LinkRun{
			URL:      string(node.Destination),
			Children: children,
		}}
	case *ast.Image:
		// Images have no Bike equivalent — render as plain text
		alt := string(node.Text(c.source))
		url := string(node.Destination)
		return []bike.InlineNode{bike.TextRun{Text: fmt.Sprintf("![%s](%s)", alt, url)}}
	case *ast.AutoLink:
		url := string(node.URL(c.source))
		return []bike.InlineNode{bike.LinkRun{
			URL:      url,
			Children: []bike.InlineNode{bike.TextRun{Text: url}},
		}}
	case *extast.Strikethrough:
		children := c.extractInlines(node)
		return []bike.InlineNode{bike.StrikethroughRun{Children: children}}
	case *extast.TaskCheckBox:
		// TaskCheckBox is handled at the list item level; skip here
		return nil
	case *ast.RawHTML:
		var text strings.Builder
		for i := 0; i < node.Segments.Len(); i++ {
			seg := node.Segments.At(i)
			text.Write(seg.Value(c.source))
		}
		return []bike.InlineNode{bike.TextRun{Text: text.String()}}
	default:
		// Check for highlight extension node
		if node, ok := n.(*highlight.Highlight); ok {
			children := c.extractInlines(node)
			return []bike.InlineNode{bike.MarkRun{Children: children}}
		}
		// Unknown inline — try to extract children
		if n.HasChildren() {
			return c.extractInlines(n)
		}
		return nil
	}
}
