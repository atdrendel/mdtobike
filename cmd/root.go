package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atdrendel/bikemark/internal/bike"
	"github.com/atdrendel/bikemark/internal/convert"
	"github.com/atdrendel/bikemark/internal/markdown"
	"github.com/spf13/cobra"
)

// convertOptions holds the options for the root convert command.
type convertOptions struct {
	markdown bool   // --markdown / -m: force treat input as Markdown
	bike     bool   // --bike / -b: force treat input as Bike
	filename string // filename for extension-based detection
}

// inputFormat represents the detected input format.
type inputFormat int

const (
	formatMarkdown inputFormat = iota
	formatBike
)

var (
	markdownFlag bool
	bikeFlag     bool
)

var rootCmd = &cobra.Command{
	Use:   "bikemark [file]",
	Short: "Convert between Markdown and Bike outline format",
	Long: `bikemark converts between GitHub-flavored Markdown and Bike outline format (.bike).

Bike is a macOS outliner that uses an HTML-based file format. bikemark
auto-detects the input format and converts to the other format.

Convert Markdown to Bike:
  bikemark input.md > output.bike

Convert Bike to Markdown:
  bikemark input.bike > output.md

Read from stdin:
  cat input.md | bikemark > output.bike
  echo "# Hello" | bikemark

Force format with flags:
  bikemark --markdown input.bike    # treat input as Markdown
  bikemark --bike input.md          # treat input as Bike`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		opts := convertOptions{
			markdown: markdownFlag,
			bike:     bikeFlag,
		}

		var input io.Reader
		if len(args) == 1 {
			opts.filename = args[0]
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()
			input = f
		} else {
			input = os.Stdin
		}

		return runConvert(input, cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// NewRootCmd returns the root command for testing purposes.
func NewRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.Flags().BoolVarP(&markdownFlag, "markdown", "m", false, "force treat input as Markdown")
	rootCmd.Flags().BoolVarP(&bikeFlag, "bike", "b", false, "force treat input as Bike")
	rootCmd.MarkFlagsMutuallyExclusive("markdown", "bike")
}

// runConvert is the testable implementation of the conversion.
func runConvert(input io.Reader, stdout, stderr io.Writer, opts convertOptions) error {
	source, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	format := detectFormat(opts, source)

	switch format {
	case formatBike:
		doc, err := bike.Parse(bytes.NewReader(source))
		if err != nil {
			return fmt.Errorf("failed to parse bike: %w", err)
		}
		return markdown.Render(stdout, doc)
	default:
		doc, err := convert.FromMarkdown(source)
		if err != nil {
			return fmt.Errorf("failed to convert markdown: %w", err)
		}
		return doc.Render(stdout)
	}
}

// detectFormat determines the input format based on flags, filename extension, and content.
// Detection priority: flags > file extension > content sniffing.
func detectFormat(opts convertOptions, content []byte) inputFormat {
	// 1. Flags (highest priority)
	if opts.markdown {
		return formatMarkdown
	}
	if opts.bike {
		return formatBike
	}

	// 2. File extension
	if opts.filename != "" {
		ext := strings.ToLower(filepath.Ext(opts.filename))
		switch ext {
		case ".bike":
			return formatBike
		case ".md", ".markdown":
			return formatMarkdown
		}
	}

	// 3. Content sniffing
	trimmed := bytes.TrimSpace(content)
	if bytes.HasPrefix(trimmed, []byte("<?xml")) {
		return formatBike
	}

	return formatMarkdown
}
