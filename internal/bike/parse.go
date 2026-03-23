package bike

import (
	"encoding/xml"
	"fmt"
	"io"
)

// Parse reads Bike XHTML from r and returns a Document.
func Parse(r io.Reader) (*Document, error) {
	decoder := xml.NewDecoder(r)
	// Bike files use HTML entities that aren't valid XML.
	// encoding/xml handles standard XML entities; HTML-specific ones
	// shouldn't appear in well-formed Bike files.

	rootID, err := findRootUL(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to find root <ul>: %w", err)
	}

	rows, err := parseRows(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rows: %w", err)
	}

	return &Document{RootID: rootID, Rows: rows}, nil
}

// findRootUL advances the decoder past <body> to the first <ul> and returns its id.
func findRootUL(decoder *xml.Decoder) (string, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("unexpected end of document: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "ul" {
			return attrValue(se.Attr, "id"), nil
		}
	}
}

// parseRows parses <li> elements within a <ul> until the closing </ul>.
func parseRows(decoder *xml.Decoder) ([]*Row, error) {
	var rows []*Row
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("unexpected end of rows: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "li" {
				row, err := parseRow(decoder, t)
				if err != nil {
					return nil, err
				}
				rows = append(rows, row)
			}
		case xml.EndElement:
			if t.Name.Local == "ul" {
				return rows, nil
			}
		}
	}
}

// parseRow parses a single <li> element (already opened) into a Row.
func parseRow(decoder *xml.Decoder, start xml.StartElement) (*Row, error) {
	row := &Row{
		ID:     attrValue(start.Attr, "id"),
		DoneAt: attrValue(start.Attr, "data-done"),
		Type:   RowType(attrValue(start.Attr, "data-type")),
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("unexpected end of row %q: %w", row.ID, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				content, err := parseParagraph(decoder)
				if err != nil {
					return nil, err
				}
				row.Content = content
			case "ul":
				children, err := parseRows(decoder)
				if err != nil {
					return nil, err
				}
				row.Children = children
			}
		case xml.EndElement:
			if t.Name.Local == "li" {
				return row, nil
			}
			// Self-closing <p/> is delivered as StartElement + EndElement by encoding/xml.
			// The EndElement for "p" here means the <p/> was self-closing and had no content.
			if t.Name.Local == "p" {
				// Content stays nil (empty row)
			}
		}
	}
}

// parseParagraph parses inline content within a <p> element.
// Returns nil for empty paragraphs (self-closing <p/>).
func parseParagraph(decoder *xml.Decoder) ([]InlineNode, error) {
	nodes, err := parseInlineContent(decoder, "p")
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes, nil
}

// parseInlineContent parses inline nodes until the closing tag with the given name.
func parseInlineContent(decoder *xml.Decoder, endTag string) ([]InlineNode, error) {
	var nodes []InlineNode
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("unexpected end of <%s>: %w", endTag, err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			text := string(t)
			if text != "" {
				nodes = append(nodes, TextRun{Text: text})
			}
		case xml.StartElement:
			node, err := parseInlineElement(decoder, t)
			if err != nil {
				return nil, err
			}
			if node != nil {
				nodes = append(nodes, node)
			}
		case xml.EndElement:
			if t.Name.Local == endTag {
				return nodes, nil
			}
		}
	}
}

// parseInlineElement parses a single inline element (already opened) into an InlineNode.
func parseInlineElement(decoder *xml.Decoder, start xml.StartElement) (InlineNode, error) {
	switch start.Name.Local {
	case "span":
		// Strip <span> — collect children as if they were direct content
		children, err := parseInlineContent(decoder, "span")
		if err != nil {
			return nil, err
		}
		// Span should contain only text; flatten to a single TextRun
		text := inlineText(children)
		return TextRun{Text: text}, nil

	case "strong":
		children, err := parseInlineContent(decoder, "strong")
		if err != nil {
			return nil, err
		}
		return StrongRun{Children: children}, nil

	case "em":
		children, err := parseInlineContent(decoder, "em")
		if err != nil {
			return nil, err
		}
		return EmRun{Children: children}, nil

	case "code":
		children, err := parseInlineContent(decoder, "code")
		if err != nil {
			return nil, err
		}
		return CodeRun{Text: inlineText(children)}, nil

	case "a":
		url := attrValue(start.Attr, "href")
		children, err := parseInlineContent(decoder, "a")
		if err != nil {
			return nil, err
		}
		return LinkRun{URL: url, Children: children}, nil

	case "s":
		children, err := parseInlineContent(decoder, "s")
		if err != nil {
			return nil, err
		}
		return StrikethroughRun{Children: children}, nil

	case "mark":
		children, err := parseInlineContent(decoder, "mark")
		if err != nil {
			return nil, err
		}
		return MarkRun{Children: children}, nil

	default:
		// Unknown element — skip to its end and treat text content as plain text
		children, err := parseInlineContent(decoder, start.Name.Local)
		if err != nil {
			return nil, err
		}
		text := inlineText(children)
		if text != "" {
			return TextRun{Text: text}, nil
		}
		return nil, nil
	}
}

// inlineText extracts the concatenated text from a slice of InlineNodes.
func inlineText(nodes []InlineNode) string {
	var s string
	for _, n := range nodes {
		switch v := n.(type) {
		case TextRun:
			s += v.Text
		case CodeRun:
			s += v.Text
		}
	}
	return s
}

// attrValue returns the value of the named attribute, or "" if not found.
func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
