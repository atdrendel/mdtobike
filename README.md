# mdtobike

Convert GitHub-flavored Markdown to [Bike](https://www.hogbaysoftware.com/bike/) outline format.

## Installation

### From source

```bash
go install github.com/atdrendel/mdtobike@latest
```

### Build locally

```bash
git clone https://github.com/atdrendel/mdtobike.git
cd mdtobike
go build .
```

## Usage

```bash
# Convert a file
mdtobike input.md > output.bike

# Convert from stdin
cat input.md | mdtobike > output.bike

# Show help
mdtobike --help

# Show version
mdtobike version
```

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
source <(mdtobike completion bash)

# Zsh
mdtobike completion zsh > "${fpath[1]}/_mdtobike"

# Fish
mdtobike completion fish | source
```

## Development

```bash
# Build
go build .

# Run tests
go test ./...

# Build with version info
go build -ldflags "-X github.com/atdrendel/mdtobike/internal/version.Version=1.0.0 \
  -X github.com/atdrendel/mdtobike/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/atdrendel/mdtobike/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" .
```

## Project Structure

```
mdtobike/
├── main.go              # Entry point
├── cmd/                 # Cobra commands
│   ├── root.go          # Root command (conversion)
│   ├── errors.go        # Sentinel errors
│   ├── version.go       # Version subcommand
│   └── completion.go    # Shell completion
├── internal/
│   ├── bike/            # Bike format types and rendering
│   └── version/         # Build-time version info
├── CLAUDE.md            # Development guidelines
└── README.md
```
