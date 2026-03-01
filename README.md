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
├── bin/                 # Development scripts
│   ├── test.sh          # Run tests
│   ├── build.sh         # Build with version info
│   ├── format.sh        # Format code
│   └── install.sh       # Build and install to /usr/local/bin
├── CLAUDE.md            # Development guidelines
└── README.md
```
