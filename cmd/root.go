package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/atdrendel/mdtobike/internal/convert"
	"github.com/spf13/cobra"
)

// convertOptions holds the options for the root convert command.
type convertOptions struct {
	// Future options will go here as flags are added
}

var rootCmd = &cobra.Command{
	Use:   "mdtobike [file]",
	Short: "Convert GitHub-flavored Markdown to Bike outline format",
	Long: `mdtobike converts GitHub-flavored Markdown files to Bike outline format (.bike).

Bike is a macOS outliner that uses an HTML-based file format. mdtobike parses
Markdown and generates properly structured Bike outlines with appropriate
row types, inline formatting, and hierarchy.

Read from a file:
  mdtobike input.md > output.bike

Read from stdin:
  cat input.md | mdtobike > output.bike
  echo "# Hello" | mdtobike`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		var input io.Reader
		if len(args) == 1 {
			f, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()
			input = f
		} else {
			input = os.Stdin
		}

		opts := convertOptions{}
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
}

// runConvert is the testable implementation of the conversion.
func runConvert(input io.Reader, stdout, stderr io.Writer, opts convertOptions) error {
	source, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	doc, err := convert.FromMarkdown(source)
	if err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}
	return doc.Render(stdout)
}
