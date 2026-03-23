# bikemark Development Guidelines

This document defines conventions for developing the bikemark CLI. Follow these guidelines to ensure consistency with Unix CLI best practices.

## Core Philosophy

1. **Do one thing well** — Convert losslessly between Markdown and Bike format
2. **Composability** — Read from stdin or file, write to stdout; pipeable by default
3. **Least surprise** — Behave like other Unix tools users already know
4. **Silence is golden** — Don't output unnecessary information on success
5. **Fail fast and loud** — Report errors immediately with clear messages
6. **Test-driven development** — Always write failing tests first, then implement

## Command Structure

bikemark is a **single-command CLI**. The root command performs bidirectional conversion between Markdown and Bike format, auto-detecting the input format:

```bash
# File argument — auto-detect direction from extension:
bikemark input.md > output.bike       # Markdown → Bike
bikemark input.bike > output.md       # Bike → Markdown

# Stdin — auto-detect from content:
cat input.md | bikemark > output.bike
cat input.bike | bikemark > output.md
echo "# Hello" | bikemark             # inline (detected as Markdown)

# Explicit flags override auto-detection:
bikemark -m input.bike                # force treat input as Markdown
bikemark -b input.md                  # force treat input as Bike
cat ambiguous | bikemark --markdown   # stdin with explicit format
```

The only subcommands are `version` and `completion`.

### Format Detection

Detection priority (first match wins):

1. **Flags**: `--markdown / -m` or `--bike / -b` (mutually exclusive — error if both)
2. **File extension**: `.md` / `.markdown` = Markdown, `.bike` = Bike
3. **Content sniffing**: starts with `<?xml` = Bike, otherwise Markdown
4. **Error**: if still ambiguous after all checks

### Input

- **File argument**: `bikemark input.md` — reads the file
- **stdin**: `cat input.md | bikemark` — reads from stdin when no argument given
- Accepts 0 or 1 positional arguments (`cobra.MaximumNArgs(1)`)
- **`--markdown / -m`**: force treat input as Markdown (bool, optional)
- **`--bike / -b`**: force treat input as Bike (bool, optional)

### Output

- **stdout**: Bike format OR Markdown, depending on detected input format
- **stderr**: Errors, warnings, progress messages

## Output Conventions

| Stream | Use for |
|--------|---------|
| stdout | Converted output — valid Bike HTML or valid Markdown |
| stderr | Errors, warnings — human-readable |

The output must be valid for its format: Bike output must open correctly in Bike.app, Markdown output must be well-formed GFM. Never mix prose with the converted output on stdout.

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
| 2 | Misuse of command (bad flags, too many args, both --markdown and --bike) |

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
    markdown bool // --markdown / -m flag
    bike     bool // --bike / -b flag
    filename string // filename for extension-based detection
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
        opts       convertOptions
        wantOutput string
        wantErr    bool
    }{
        {
            name:       "markdown to bike: heading",
            input:      "# Hello",
            wantOutput: `<li id="`,  // partial match
        },
        {
            name:       "bike to markdown: heading",
            input:      `<?xml version="1.0" encoding="UTF-8"?>...`,
            wantOutput: "# ",  // partial match
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            input := strings.NewReader(tt.input)
            stdout := new(bytes.Buffer)
            stderr := new(bytes.Buffer)

            err := runConvert(input, stdout, stderr, tt.opts)

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
- Valid Bike input produces valid Markdown output
- Each row type maps correctly in both directions
- Inline formatting (bold, italic, code, links) in both directions
- Nesting/hierarchy: heading structure (flat→tree and tree→flat)
- Empty input
- Stdin vs file input
- Format detection: extension, content sniffing, flags
- Flag validation: error when both --markdown and --bike provided
- Error cases (file not found, invalid input)
- Round-trip: md→bike→md produces equivalent Markdown

## Code Organization

```
bikemark/
├── main.go                    # Entry point with error handling
├── cmd/                       # Cobra commands
│   ├── root.go                # Root command (bidirectional conversion)
│   ├── root_test.go           # Root command tests
│   ├── convert_test.go        # End-to-end conversion tests (both directions)
│   ├── errors.go              # Sentinel errors
│   ├── version.go             # Version subcommand
│   └── completion.go          # Shell completion subcommand
├── internal/
│   ├── bike/                  # Bike document model + XHTML renderer (no Markdown knowledge)
│   │   ├── bike.go            # Types: Document, Row, InlineNode, IDGenerator
│   │   ├── render.go          # XHTML rendering with span wrapping
│   │   ├── parse.go           # Parse .bike XHTML → Document
│   │   └── bike_test.go
│   ├── convert/               # Markdown AST → Bike document transformation
│   │   ├── convert.go         # goldmark parsing + heading hierarchy + block/inline mapping
│   │   └── convert_test.go
│   ├── markdown/              # Bike document → Markdown text
│   │   ├── render.go          # Render Bike Document as Markdown
│   │   └── render_test.go
│   └── version/               # Build-time version info
│       └── version.go
├── bin/                       # Development scripts
│   ├── test.sh                # Run tests
│   ├── build.sh               # Build with version info
│   ├── format.sh              # Format code (gofmt)
│   └── install.sh             # Build and install to /usr/local/bin
├── CLAUDE.md
├── README.md
└── go.mod
```

## Development Scripts

All development commands live in `bin/` and should be run from the project root:

```bash
./bin/test.sh       # Run tests (go test ./...)
./bin/build.sh      # Build with ldflags (version/commit/date)
./bin/format.sh     # Format code (gofmt -w .)
./bin/install.sh    # Build and install to /usr/local/bin
```

Each script follows a common scaffolding pattern (from web-to-markdown-apple):
- `set -o errexit`, `nounset`, `pipefail` for strict error handling
- `TRACE=1 ./bin/test.sh` enables `set -o xtrace` for debugging
- `--help` / `-h` flag prints usage
- `pushd`/`popd` to the project root so scripts work from any directory

### Version Injection via ldflags

`build.sh` and `install.sh` inject version info automatically:

```bash
go build -ldflags "-X github.com/atdrendel/bikemark/internal/version.Version=dev \
  -X github.com/atdrendel/bikemark/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/atdrendel/bikemark/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
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
| `hr` | `---` horizontal rule | Horizontal rule; empty `<p/>`, no content |

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

The following is an annotated excerpt from a real Bike 1.x file, demonstrating headings with nested tasks, inline formatting with span wrapping, note rows, code rows, blockquotes, ordered/unordered lists, horizontal rules, empty rows, and body rows:

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

      <!-- Horizontal rule -->
      <li id="hr" data-type="hr">
        <p/>
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
| `---` | `data-type="hr"` | Horizontal rule; empty `<p/>` |
| `- item` | `data-type="unordered"` | Unordered list; nesting preserved |
| `1. item` | `data-type="ordered"` | Ordered list; nesting preserved |
| `- [ ] task` | `data-type="task"` | Uncompleted task |
| `- [x] task` | `data-type="task"` + `data-done` | Completed task |
| `![alt](url)` | Body row with source text | Images have no Bike equivalent; render as plain text |
| GFM table | Body rows (nested) | Header row as parent, data rows as children; cells joined with " — " |
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

## Bike-to-Markdown Mapping

### Block-Level Mapping

| Bike Row Type | GFM Element | Notes |
|---|---|---|
| `data-type="heading"` | `#` heading | Nesting depth determines heading level (depth 0 = `#`, depth 1 = `##`, etc.) |
| *(no data-type)* / body | Paragraph | Plain paragraph |
| `data-type="quote"` | `>` blockquote | Sibling quote rows become lines in one blockquote |
| `data-type="code"` | ```` ``` ```` fenced code | Consecutive code siblings merge into one fenced block |
| `data-type="hr"` | `---` | Horizontal rule |
| `data-type="unordered"` | `- item` | Unordered list; nesting preserved |
| `data-type="ordered"` | `1. item` | Ordered list; nesting preserved |
| `data-type="task"` (no `data-done`) | `- [ ] task` | Uncompleted task |
| `data-type="task"` + `data-done` | `- [x] task` | Completed task |
| `data-type="note"` | Paragraph | Lossy — no Markdown equivalent for note rows |

### Heading Hierarchy (Tree-to-Flat Conversion)

The reverse of flat-to-tree. Bike nesting depth determines the Markdown heading level:

- Top-level heading row → `# H1`
- Heading nested 1 deep → `## H2`
- Heading nested 2 deep → `### H3`
- Non-heading children of a heading → rendered after the heading (as paragraphs, lists, etc.)

### Inline Mapping

| Bike HTML | GFM |
|---|---|
| `<strong>` | `**bold**` |
| `<em>` | `*italic*` |
| `<code>` | `` `code` `` |
| `<a href="url">text</a>` | `[text](url)` |
| `<s>` | `~~strike~~` |
| `<mark>` | `==highlight==` |
| `<span>` | *(stripped — content becomes plain text)* |

## Losslessness

The goal is lossless round-trip conversion to the greatest extent possible. Known lossy areas:

### Markdown → Bike (lossy)

- **Heading levels**: all become `heading` row type; level is inferred from nesting depth (recoverable on round-trip)
- **Image syntax**: `![alt](url)` becomes plain text body row (not recoverable)
- **Code fence language**: `` ```go `` becomes code rows with no language info (not recoverable)
- **Table structure**: GFM tables become body rows with " — " cell separators (heuristically recoverable)

### Bike → Markdown (lossy)

- **Row IDs**: every `<li>` has a unique ID; Markdown has no equivalent (not recoverable)
- **Note rows**: `data-type="note"` has no Markdown equivalent; converted to paragraph (not recoverable as note)
- **Task timestamps**: `data-done="2026-02-16T14:30:47Z"` becomes `[x]`; exact timestamp lost

### Round-trip safe

These elements survive md→bike→md and bike→md→bike without information loss:
- Paragraphs (body rows)
- Bold, italic, inline code, links, strikethrough, highlight
- Ordered and unordered lists with nesting
- Blockquotes
- Horizontal rules
- Tasks (checked/unchecked status)
- Heading hierarchy (level preserved via nesting depth)

## Conversion Pipeline Architecture

The tool supports two conversion directions, sharing a common Bike document model.

### Markdown → Bike (existing)

Three phases:
1. **Parse** (Markdown → AST): Read Markdown, produce goldmark AST
2. **Transform** (AST → Bike Document Model): Walk AST, build heading hierarchy, map blocks/inlines
3. **Render** (Bike Model → XHTML): Serialize to `.bike` format with IDs, indentation, span wrapping

### Bike → Markdown (to be implemented)

Three phases:
1. **Parse** (XHTML → Bike Document Model): Parse `.bike` XML, extract rows, types, inline content
2. **Transform** (Bike Model → Markdown IR): Flatten heading tree, merge code rows, map types
3. **Render** (Markdown IR → Text): Serialize to GFM text

### Package separation

- `internal/bike/` — Bike document model types, XHTML renderer, and XHTML parser (shared by both directions)
- `internal/convert/` — Markdown AST → Bike document transformation (uses goldmark)
- `internal/markdown/` — Bike document → Markdown text rendering
- `cmd/` — Orchestrates the pipeline: detect format, wire appropriate parse → transform → render

## Checklist for Changes

- [ ] `--help` works and is accurate
- [ ] Errors go to stderr with clear messages
- [ ] Returns correct exit codes (0 success, 1 error, 2 misuse)
- [ ] Works in pipes (non-TTY mode)
- [ ] Has unit tests
- [ ] Output is valid Bike format that opens in Bike.app (when converting to Bike)
- [ ] Output is valid Markdown (when converting from Bike)
- [ ] Round-trip test: md→bike→md produces equivalent Markdown
- [ ] **`cmd.SilenceUsage = true`** in `RunE` for the root command
- [ ] **Simulate the full user experience**: mentally run the command and read the complete output — check for redundant messages, unclear feedback, or missing information
