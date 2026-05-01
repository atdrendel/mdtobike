# bikemark

Convert between GitHub-flavored Markdown and [Bike](https://www.hogbaysoftware.com/bike/) outline format.

## Installation

### From source

```bash
go install github.com/atdrendel/bikemark@latest
```

### Build locally

```bash
git clone https://github.com/atdrendel/bikemark.git
cd bikemark
go build .
```

## Usage

```bash
# Convert Markdown to Bike
bikemark input.md > output.bike

# Convert Bike to Markdown
bikemark input.bike > output.md

# Convert from stdin
cat input.md | bikemark > output.bike

# Show help
bikemark --help

# Show version
bikemark version
```

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
source <(bikemark completion bash)

# Zsh
bikemark completion zsh > "${fpath[1]}/_bikemark"

# Fish
bikemark completion fish | source
```

## Development

```bash
# Run tests
./bin/test.sh

# Build with version info
./bin/build.sh

# Format code
./bin/format.sh

# Install to /usr/local/bin
./bin/install.sh
```

## Project Structure

```
bikemark/
├── main.go              # Entry point
├── cmd/                 # Cobra commands
│   ├── root.go          # Root command (conversion)
│   ├── errors.go        # Sentinel errors
│   ├── version.go       # Version subcommand
│   └── completion.go    # Shell completion
├── internal/
│   ├── bike/            # Bike format types and rendering
│   ├── convert/         # Markdown → Bike conversion
│   ├── markdown/        # Bike → Markdown conversion
│   └── version/         # Build-time version info
├── bin/                 # Development scripts
│   ├── test.sh          # Run tests
│   ├── build.sh         # Build with version info
│   ├── format.sh        # Format code
│   └── install.sh       # Build and install to /usr/local/bin
├── AGENTS.md            # Development guidelines
└── README.md
```
