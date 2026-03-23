package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunConvertMarkdownToBike(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "heading produces valid bike",
			input:      "# Hello",
			wantOutput: `data-type="heading"`,
		},
		{
			name:       "xml declaration present",
			input:      "# Test",
			wantOutput: `<?xml version="1.0" encoding="UTF-8"?>`,
		},
		{
			name:       "paragraph produces body row",
			input:      "Just text",
			wantOutput: `<p>Just text</p>`,
		},
		{
			name:       "empty input produces valid document",
			input:      "",
			wantOutput: `<html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			opts := convertOptions{}

			err := runConvert(input, stdout, stderr, opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("runConvert() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := stdout.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output does not contain %q\ngot:\n%s", tt.wantOutput, output)
			}
		})
	}
}

// minimalBikeDoc wraps row XML in a valid Bike document for testing.
func minimalBikeDoc(rows string) string {
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

func TestRunConvertBikeToMarkdown(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "heading produces markdown heading",
			input: minimalBikeDoc(`      <li id="a" data-type="heading">
        <p>Hello</p>
      </li>`),
			wantOutput: "# Hello",
		},
		{
			name: "body row produces paragraph",
			input: minimalBikeDoc(`      <li id="a">
        <p>Just text</p>
      </li>`),
			wantOutput: "Just text",
		},
		{
			name: "code rows produce fenced block",
			input: minimalBikeDoc(`      <li id="a" data-type="code">
        <p>line 1</p>
      </li>
      <li id="b" data-type="code">
        <p>line 2</p>
      </li>`),
			wantOutput: "```\nline 1\nline 2\n```",
		},
		{
			name: "task row produces checkbox",
			input: minimalBikeDoc(`      <li id="a" data-type="task">
        <p>Todo</p>
      </li>`),
			wantOutput: "- [ ] Todo",
		},
		{
			name: "checked task produces checked checkbox",
			input: minimalBikeDoc(`      <li id="a" data-done="2026-01-01T00:00:00Z" data-type="task">
        <p>Done</p>
      </li>`),
			wantOutput: "- [x] Done",
		},
		{
			name:       "empty document produces empty output",
			input:      minimalBikeDoc(""),
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)
			opts := convertOptions{}

			err := runConvert(input, stdout, stderr, opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("runConvert() error = %v, wantErr %v", err, tt.wantErr)
			}

			output := stdout.String()
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("output does not contain %q\ngot:\n%s", tt.wantOutput, output)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		opts    convertOptions
		content string
		want    inputFormat
	}{
		{
			name:    "markdown flag overrides everything",
			opts:    convertOptions{markdown: true, filename: "test.bike"},
			content: `<?xml version="1.0"?>`,
			want:    formatMarkdown,
		},
		{
			name:    "bike flag overrides everything",
			opts:    convertOptions{bike: true, filename: "test.md"},
			content: "# Hello",
			want:    formatBike,
		},
		{
			name:    "md extension",
			opts:    convertOptions{filename: "test.md"},
			content: "anything",
			want:    formatMarkdown,
		},
		{
			name:    "markdown extension",
			opts:    convertOptions{filename: "notes.markdown"},
			content: "anything",
			want:    formatMarkdown,
		},
		{
			name:    "bike extension",
			opts:    convertOptions{filename: "outline.bike"},
			content: "anything",
			want:    formatBike,
		},
		{
			name:    "bike extension case insensitive",
			opts:    convertOptions{filename: "outline.BIKE"},
			content: "anything",
			want:    formatBike,
		},
		{
			name:    "content sniff xml prefix",
			opts:    convertOptions{},
			content: `<?xml version="1.0" encoding="UTF-8"?>`,
			want:    formatBike,
		},
		{
			name:    "content sniff xml with leading whitespace",
			opts:    convertOptions{},
			content: "  \n<?xml version=\"1.0\"?>",
			want:    formatBike,
		},
		{
			name:    "content sniff markdown heading",
			opts:    convertOptions{},
			content: "# Hello",
			want:    formatMarkdown,
		},
		{
			name:    "content sniff plain text defaults to markdown",
			opts:    convertOptions{},
			content: "just some text",
			want:    formatMarkdown,
		},
		{
			name:    "unknown extension falls through to content sniff",
			opts:    convertOptions{filename: "notes.txt"},
			content: `<?xml version="1.0"?>`,
			want:    formatBike,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(tt.opts, []byte(tt.content))
			if got != tt.want {
				t.Errorf("detectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunConvertWithFormatFlags(t *testing.T) {
	// Force markdown flag on XML content — should treat as markdown, not bike
	bikeContent := minimalBikeDoc(`      <li id="a" data-type="heading">
        <p>Hello</p>
      </li>`)

	t.Run("markdown flag forces markdown parsing", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		opts := convertOptions{markdown: true}

		err := runConvert(strings.NewReader(bikeContent), stdout, stderr, opts)
		if err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}
		// When treated as markdown, the XML becomes bike output (not markdown output)
		output := stdout.String()
		if !strings.Contains(output, "<?xml") {
			t.Errorf("expected bike output when forcing markdown input, got:\n%s", output)
		}
	})

	t.Run("bike flag forces bike parsing", func(t *testing.T) {
		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		opts := convertOptions{bike: true}

		err := runConvert(strings.NewReader(bikeContent), stdout, stderr, opts)
		if err != nil {
			t.Fatalf("runConvert() error = %v", err)
		}
		// When treated as bike, should produce markdown output
		output := stdout.String()
		if !strings.Contains(output, "# Hello") {
			t.Errorf("expected markdown output when forcing bike input, got:\n%s", output)
		}
	})
}

func TestRoundTripMarkdownToBikeToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string // expected substring in round-tripped output
	}{
		{"paragraph", "Hello world", "Hello world"},
		{"heading", "# Title", "# Title"},
		{"heading hierarchy", "# Top\n\n## Sub\n\nContent", "# Top"},
		{"heading hierarchy sub", "# Top\n\n## Sub\n\nContent", "## Sub"},
		{"heading hierarchy content", "# Top\n\n## Sub\n\nContent", "Content"},
		{"bold", "**bold**", "**bold**"},
		{"italic", "*italic*", "*italic*"},
		{"inline code", "`code`", "`code`"},
		{"link", "[text](https://example.com)", "[text](https://example.com)"},
		{"strikethrough", "~~strike~~", "~~strike~~"},
		{"highlight", "==highlight==", "==highlight=="},
		{"unordered list", "- One\n- Two", "- One\n- Two"},
		{"ordered list", "1. First\n2. Second", "1. First\n1. Second"},
		{"task unchecked", "- [ ] Todo", "- [ ] Todo"},
		{"task checked", "- [x] Done", "- [x] Done"},
		{"blockquote", "> Quote", "> Quote"},
		{"code block", "```\ncode\n```", "```\ncode\n```"},
		{"horizontal rule", "---", "---"},
		{"nested list", "- Parent\n  - Child", "- Parent\n  - Child"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Markdown → Bike
			mdToBike := new(bytes.Buffer)
			err := runConvert(strings.NewReader(tt.markdown), mdToBike, new(bytes.Buffer), convertOptions{})
			if err != nil {
				t.Fatalf("md→bike error: %v", err)
			}

			// Bike → Markdown
			bikeToMd := new(bytes.Buffer)
			err = runConvert(strings.NewReader(mdToBike.String()), bikeToMd, new(bytes.Buffer), convertOptions{})
			if err != nil {
				t.Fatalf("bike→md error: %v", err)
			}

			output := bikeToMd.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("round-trip output does not contain %q\ngot:\n%s", tt.want, output)
			}
		})
	}
}
