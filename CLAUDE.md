# mdtobike Development Guidelines

This document defines conventions for developing the mdtobike CLI. Follow these guidelines to ensure consistency with Unix CLI best practices.

## Core Philosophy

1. **Do one thing well** — Convert Markdown to Bike format, and do it correctly
2. **Composability** — Read from stdin or file, write to stdout; pipeable by default
3. **Least surprise** — Behave like other Unix tools users already know
4. **Silence is golden** — Don't output unnecessary information on success
5. **Fail fast and loud** — Report errors immediately with clear messages
6. **Test-driven development** — Always write failing tests first, then implement

## Command Structure

mdtobike is a **single-command CLI** (not a subcommand-based CRUD tool like ankigo). The root command performs the conversion:

```bash
mdtobike input.md > output.bike       # file argument
cat input.md | mdtobike > output.bike  # stdin
echo "# Hello" | mdtobike             # inline
```

The only subcommands are `version` and `completion`.

### Input

- **File argument**: `mdtobike input.md` — reads the file
- **stdin**: `cat input.md | mdtobike` — reads from stdin when no argument given
- Accepts 0 or 1 positional arguments (`cobra.MaximumNArgs(1)`)

### Output

- **stdout**: Bike format output (the converted document)
- **stderr**: Errors, warnings, progress messages

## Output Conventions

| Stream | Use for |
|--------|---------|
| stdout | Bike format output — must be valid Bike HTML |
| stderr | Errors, warnings — human-readable |

The output must be valid Bike format that opens correctly in Bike.app. Never mix prose with the Bike output on stdout.

## Error Handling

### Error Message Format

```
Error: <what went wrong>
```

With context:
```
Error: failed to open file: no such file or directory
```

### Sentinel Errors

Defined in `cmd/errors.go`:

- `ErrCancelled`: User cancelled an operation. Exit 1, no additional message.
- `ErrSilent`: Command failed but already printed specific error messages. Exit 1, no additional message.

Before adding new error handling, check `cmd/errors.go` for existing patterns.

### Silence Usage on Runtime Errors

Cobra prints usage/help when `RunE` returns an error. This is correct for argument validation errors but NOT for runtime errors.

Add `cmd.SilenceUsage = true` at the start of `RunE` for the root command (already done):

```go
RunE: func(cmd *cobra.Command, args []string) error {
    cmd.SilenceUsage = true
    // ... rest of implementation
},
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (file not found, parse error, conversion error) |
| 2 | Misuse of command (bad flags, too many args) |

## Testing Requirements

Use TDD. For each feature:

1. Write failing tests first
2. Implement to pass tests
3. Refactor if needed

### Testable Function Signatures

All command logic lives in testable functions that accept `io.Reader`/`io.Writer`, not in Cobra command handlers directly:

```go
// Options struct holds flags
type convertOptions struct {
    // flags go here
}

// Cobra command extracts flags and calls the testable function
RunE: func(cmd *cobra.Command, args []string) error {
    opts := convertOptions{}
    return runConvert(input, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
}

// Testable function — all business logic here
func runConvert(input io.Reader, stdout, stderr io.Writer, opts convertOptions) error {
    // implementation
}
```

### Test Patterns

```go
// Table-driven tests
func TestRunConvert(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        wantOutput string
        wantErr    bool
    }{
        {
            name:       "heading",
            input:      "# Hello",
            wantOutput: `<li id="`,  // partial match
            wantErr:    false,
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
            if tt.wantOutput != "" && !strings.Contains(stdout.String(), tt.wantOutput) {
                t.Errorf("output = %q, want to contain %q", stdout.String(), tt.wantOutput)
            }
        })
    }
}
```

### What to Test

- Valid Markdown input produces valid Bike output
- Each row type maps correctly (headings, lists, code blocks, etc.)
- Inline formatting (bold, italic, code, links)
- Nesting/hierarchy from heading structure
- Empty input
- Stdin vs file input
- Error cases (file not found, invalid input)

## Code Organization

```
mdtobike/
├── main.go                    # Entry point with error handling
├── cmd/                       # Cobra commands
│   ├── root.go                # Root command (the convert command)
│   ├── root_test.go           # Root command tests
│   ├── convert_test.go        # End-to-end conversion tests
│   ├── errors.go              # Sentinel errors
│   ├── version.go             # Version subcommand
│   └── completion.go          # Shell completion subcommand
├── internal/
│   ├── bike/                  # Bike document model + XHTML renderer (no Markdown knowledge)
│   │   ├── bike.go            # Types: Document, Row, InlineNode, IDGenerator
│   │   ├── render.go          # XHTML rendering with span wrapping
│   │   └── bike_test.go
│   ├── convert/               # Markdown AST → Bike document transformation
│   │   ├── convert.go         # goldmark parsing + heading hierarchy + block/inline mapping
│   │   └── convert_test.go
│   └── version/               # Build-time version info
│       └── version.go
├── CLAUDE.md
├── README.md
└── go.mod
```

## Build and Version Management

### Version Injection via ldflags

```bash
go build -ldflags "-X github.com/atdrendel/mdtobike/internal/version.Version=1.0.0 \
  -X github.com/atdrendel/mdtobike/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/atdrendel/mdtobike/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
```

### Development Build

```bash
go build .
```

### Running Tests

```bash
go test ./...
```

## Bike Format Specification

Bike is a macOS outliner app by Jesse Grosjean (Hog Bay Software). Bike 1.x uses an HTML-based file format.

### Document Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<html>
  <head>
    <meta charset="utf-8"/>
  </head>
  <body>
    <ul id="ROOT_ID">
      <li id="ROW_ID" data-type="heading">
        <p>Row content here</p>
        <ul>
          <li id="CHILD_ID">
            <p>Child content</p>
          </li>
        </ul>
      </li>
    </ul>
  </body>
</html>
```

Key structural rules:
- XML declaration: `<?xml version="1.0" encoding="UTF-8"?>`
- Root element: `<html>` (no xmlns namespace)
- Self-closing tags where appropriate: `<meta charset="utf-8"/>`
- Single `<ul>` in `<body>` is the root list, with a unique `id`
- Each row is a `<li>` with a unique `id`
- Content always inside `<p>` within each `<li>`
- Nesting: `<ul>` inside `<li>`, placed after the `<p>`
- Indentation: 2 spaces per nesting level
- Attribute order on `<li>`: `id`, then `data-done` (if present), then `data-type` (if present)

### Row IDs

- Each `<li>` has a unique `id` attribute
- IDs are short alphanumeric strings (typically 2-3 characters)
- Characters: letters (upper/lowercase), digits, hyphens, underscores
- The root `<ul>` has a longer ID (typically 8 characters)
- IDs must be unique within the document

### Row Types (`data-type` attribute)

Verified against real Bike 1.x output:

| `data-type` value | Markdown equivalent | Notes |
|---|---|---|
| *(none)* | Plain paragraph | Default "body" row type — no `data-type` attribute present |
| `heading` | `#`, `##`, etc. | Single heading type; level determined by outline depth |
| `quote` | `>` blockquote | One row per line; multi-line blockquotes become sibling quote rows |
| `code` | ```` ``` ```` fenced code | **One row per line of code** — each line is a separate `<li data-type="code">` sibling |
| `note` | *(none)* | Bike-specific; no Markdown equivalent |
| `ordered` | `1.`, `2.`, etc. | Ordered list item; nesting via `<ul>` inside parent `<li>` |
| `unordered` | `-`, `*`, `+` | Unordered list item; nesting via `<ul>` inside parent `<li>` |
| `task` | `- [ ]`, `- [x]` | Task/checkbox item |

**Bike does NOT support horizontal rules.** There is no `hr` row type.

### Task Completion

Completed tasks have a `data-done` attribute with an ISO 8601 UTC timestamp:

```xml
<li id="9_" data-done="2026-02-16T14:30:47Z" data-type="task">
  <p>Completed task</p>
</li>
```

Uncompleted tasks have no `data-done` attribute.

### Empty Rows

Empty rows use self-closing `<p/>`:

```xml
<li id="sd">
  <p/>
</li>
```

### Inline Formatting

Verified against real Bike 1.x output:

| HTML element | Markdown equivalent |
|---|---|
| `<strong>` | `**bold**` |
| `<em>` | `*italic*` |
| `<code>` | `` `inline code` `` |
| `<a href="url">` | `[text](url)` |
| `<s>` | `~~strikethrough~~` |
| `<mark>` | `==highlighted==` (extension syntax) |

Inline elements can nest: `<em><mark>text</mark></em>`.

### The `<span>` Wrapping Rule

This is critical to get right. Bike wraps plain text runs in `<span>` tags **only when the `<p>` also contains formatting elements**. The rule:

1. `<p>` contains **only plain text** → no `<span>`: `<p>Just plain text</p>`
2. `<p>` contains **only formatting** (no plain text) → no `<span>`: `<p><code>some_method</code></p>`
3. `<p>` contains **both plain text and formatting** → all plain text runs get `<span>`:

```xml
<!-- Leading text before formatting -->
<p><span>Did we replace </span><code>some_method</code></p>

<!-- Formatting then text -->
<p><strong>Label:</strong><span> trailing text</span></p>

<!-- Text, formatting, text (multiple spans) -->
<p><span>Can we disable </span><code>metadata.enabled</code><span> ?</span></p>

<!-- Multiple formatting elements with text between them -->
<p><strong>Size on disk:</strong><span> </span><code>17,792,600,039 bytes</code></p>
```

Note: `<span>` can contain just whitespace (e.g., `<span> </span>` between two formatting elements). Text inside formatting elements (`<strong>`, `<em>`, etc.) is never wrapped in `<span>` — only top-level text runs within the `<p>` are.

### Comprehensive Real-World Example

The following is an annotated excerpt from a real Bike 1.x file, demonstrating headings with nested tasks, inline formatting with span wrapping, note rows, code rows, blockquotes, ordered/unordered lists, empty rows, and body rows:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<html>
  <head>
    <meta charset="utf-8"/>
  </head>
  <body>
    <ul id="JSTtryh_">

      <!-- Heading with nested task children -->
      <li id="T2" data-type="heading">
        <p>Email notifications</p>
        <ul>
          <!-- Unchecked task -->
          <li id="qbz" data-type="task">
            <p>Approve pull request</p>
          </li>
          <!-- Checked task: data-done comes before data-type -->
          <li id="9_" data-done="2026-02-16T14:30:47Z" data-type="task">
            <p><span>Did we replace </span><code>should_notify?</code></p>
            <ul>
              <!-- Note row (Bike-specific, no Markdown equivalent) -->
              <li id="FI7" data-type="note">
                <p><span>We now call it in </span><code>create_workflow</code></p>
              </li>
            </ul>
          </li>
        </ul>
      </li>

      <!-- Code block: one row per line, sibling <li> elements -->
      <li id="ri" data-type="code">
        <p>func hello() {</p>
      </li>
      <li id="Ls" data-type="code">
        <p>  print("Hello")</p>
      </li>
      <li id="qy" data-type="code">
        <p>}</p>
      </li>

      <!-- Empty body row -->
      <li id="sd">
        <p/>
      </li>

      <!-- Blockquote: one row per line, with nested inline formatting -->
      <li id="Av" data-type="quote">
        <p><em><mark>Highlighted italic text</mark></em><span> and plain text.</span></p>
      </li>
      <li id="q7" data-type="quote">
        <p><span>Second line with </span><a href="https://example.com">Example</a></p>
      </li>

      <!-- Ordered list with nesting -->
      <li id="aK" data-type="ordered">
        <p><span>T</span><code>hre</code><span>e</span></p>
        <ul>
          <li id="IL" data-type="ordered">
            <p>One</p>
          </li>
          <li id="Qx" data-type="ordered">
            <p>Two</p>
          </li>
        </ul>
      </li>

      <!-- Unordered list with nesting -->
      <li id="xv" data-type="unordered">
        <p><span>T</span><strong>w</strong><span>o</span></p>
        <ul>
          <li id="0k" data-type="unordered">
            <p>One</p>
          </li>
        </ul>
      </li>

      <!-- Plain body rows (no data-type) -->
      <li id="cK">
        <p>Here's a plain text row</p>
      </li>

    </ul>
  </body>
</html>
```

## Markdown-to-Bike Mapping

### Block-Level Mapping

| GFM Element | Bike Row Type | Notes |
|---|---|---|
| `# Heading` | `data-type="heading"` | All heading levels map to `heading`; depth conveyed by outline nesting |
| `## Subheading` | `data-type="heading"` (nested) | Nested under the parent heading's `<ul>` |
| Paragraph | *(no data-type)* | Default body row |
| `> Blockquote` | `data-type="quote"` | One row per line |
| ```` ``` code ```` | `data-type="code"` | One row per line; language info dropped |
| `---` | Body row with `---` text | Bike has no HR type; render as plain text |
| `- item` | `data-type="unordered"` | Unordered list; nesting preserved |
| `1. item` | `data-type="ordered"` | Ordered list; nesting preserved |
| `- [ ] task` | `data-type="task"` | Uncompleted task |
| `- [x] task` | `data-type="task"` + `data-done` | Completed task |
| `![alt](url)` | Body row with source text | Images have no Bike equivalent; render as plain text |
| Raw HTML blocks | Body row with text content | Strip tags, keep text content |

### Heading Hierarchy (Flat-to-Tree Conversion)

Markdown is flat — headings establish sections but content isn't syntactically nested. Bike is inherently hierarchical. The converter must "lift" flat Markdown into a tree:

- `# H1` → top-level heading row
- `## H2` → nested inside the preceding H1's `<ul>`
- `### H3` → nested inside the preceding H2's `<ul>`
- Paragraphs, lists, etc. under a heading → nested inside that heading's `<ul>`

When a lower-level heading follows a higher-level heading (e.g., `## H2` followed by `# H1`), the tree "pops" back up to the appropriate level.

### Inline Mapping

| GFM | Bike HTML |
|---|---|
| `**bold**` | `<strong>bold</strong>` |
| `*italic*` | `<em>italic</em>` |
| `` `code` `` | `<code>code</code>` |
| `[text](url)` | `<a href="url">text</a>` |
| `~~strike~~` | `<s>strike</s>` |
| `==highlight==` | `<mark>highlight</mark>` |
| Plain text (adjacent to formatting) | `<span>text</span>` |

## Conversion Pipeline Architecture

The conversion follows three phases:

### 1. Parse (Markdown → AST)
Read Markdown input and produce an abstract syntax tree. This phase uses a Markdown parser library.

### 2. Transform (AST → Bike Document Model)
Walk the AST and build a Bike document model — a tree of rows with types, content, and nesting. This is where heading hierarchy is resolved (flat headings → nested tree).

### 3. Render (Bike Model → XHTML Output)
Serialize the Bike document model to the `.bike` XHTML format with proper indentation, IDs, and inline formatting.

Package separation:
- `internal/bike/` — Bike document model types and XHTML renderer (no Markdown knowledge)
- `internal/convert/` — Markdown AST → Bike document transformation (uses goldmark)
- `cmd/` — Orchestrates the pipeline, wires parse → transform → render

## Checklist for Changes

- [ ] `--help` works and is accurate
- [ ] Errors go to stderr with clear messages
- [ ] Returns correct exit codes (0 success, 1 error, 2 misuse)
- [ ] Works in pipes (non-TTY mode)
- [ ] Has unit tests
- [ ] Output is valid Bike format that opens in Bike.app
- [ ] **`cmd.SilenceUsage = true`** in `RunE` for the root command
- [ ] **Simulate the full user experience**: mentally run the command and read the complete output — check for redundant messages, unclear feedback, or missing information
