// Package bike provides types and functions for generating Bike outline format (.bike) files.
//
// The Bike format is an HTML-based outliner format used by Bike for macOS.
// It represents outlines as nested <ul>/<li> trees with data-type attributes
// for row classification and standard HTML inline elements for text formatting.
package bike

// RowType identifies the type of a Bike outline row.
type RowType string

const (
	RowTypeBody      RowType = ""          // default body row (no data-type attribute)
	RowTypeHeading   RowType = "heading"   // heading row
	RowTypeQuote     RowType = "quote"     // blockquote row
	RowTypeCode      RowType = "code"      // code row (one per line)
	RowTypeOrdered   RowType = "ordered"   // ordered list item
	RowTypeUnordered RowType = "unordered" // unordered list item
	RowTypeTask      RowType = "task"      // task/checkbox item
)

// InlineNode represents inline content within a row's <p> element.
type InlineNode interface {
	inlineNode() // sealed marker method
}

// TextRun is a plain text segment.
type TextRun struct{ Text string }

// StrongRun is bold text (<strong>).
type StrongRun struct{ Children []InlineNode }

// EmRun is italic text (<em>).
type EmRun struct{ Children []InlineNode }

// CodeRun is inline code (<code>).
type CodeRun struct{ Text string }

// LinkRun is a hyperlink (<a>).
type LinkRun struct {
	URL      string
	Children []InlineNode
}

// StrikethroughRun is strikethrough text (<s>).
type StrikethroughRun struct{ Children []InlineNode }

// MarkRun is highlighted text (<mark>).
type MarkRun struct{ Children []InlineNode }

func (TextRun) inlineNode()           {}
func (StrongRun) inlineNode()         {}
func (EmRun) inlineNode()             {}
func (CodeRun) inlineNode()           {}
func (LinkRun) inlineNode()           {}
func (StrikethroughRun) inlineNode()  {}
func (MarkRun) inlineNode()           {}

// Row is a single Bike outline row (<li>).
type Row struct {
	ID       string       // unique identifier
	Type     RowType      // row type (empty string = body)
	Content  []InlineNode // <p> content; nil = empty row (<p/>)
	Children []*Row       // nested rows (<ul>)
	DoneAt   string       // ISO 8601 UTC timestamp for completed tasks
}

// Document is a complete Bike file.
type Document struct {
	RootID string // root <ul> id
	Rows   []*Row
}

// IDGenerator produces unique short IDs for Bike rows.
type IDGenerator struct {
	counter int
}

// chars used for ID generation (letters, digits, and a few safe symbols).
const idChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Next returns the next unique row ID.
func (g *IDGenerator) Next() string {
	id := g.encode(g.counter)
	g.counter++
	return id
}

// RootID returns an 8-character ID suitable for the root <ul>.
func (g *IDGenerator) RootID() string {
	// Use a longer encoding starting from a high offset
	return g.encode(g.counter + 100000)
}

func (g *IDGenerator) encode(n int) string {
	base := len(idChars)
	if n < base {
		return string(idChars[n])
	}
	result := make([]byte, 0, 4)
	for n > 0 {
		result = append(result, idChars[n%base])
		n /= base
	}
	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}
