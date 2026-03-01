package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/atdrendel/mdtobike/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, cmd.ErrCancelled) || errors.Is(err, cmd.ErrSilent) {
			os.Exit(1)
		}
		// Don't print usage errors - Cobra already showed the usage
		if isUsageError(err) {
			os.Exit(2) // Exit code 2 for misuse per CLAUDE.md
		}
		// Print the error for other cases
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		// Unknown command is still misuse, use exit code 2
		if isMisuseError(err) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

// isUsageError returns true if the error is a Cobra usage error
// where Cobra has already printed usage text. We don't need to print
// the error message again for these cases.
// Note: "unknown command" is NOT included because Cobra doesn't
// print usage for unknown commands.
func isUsageError(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "accepts ") ||
		strings.HasPrefix(msg, "requires ") ||
		strings.HasPrefix(msg, "unknown flag") ||
		strings.HasPrefix(msg, "unknown shorthand flag") ||
		strings.Contains(msg, "flag needs an argument") ||
		strings.HasPrefix(msg, "invalid argument")
}

// isMisuseError returns true if the error indicates command misuse
// (should exit with code 2) but needs the error message printed
// because Cobra didn't show usage.
func isMisuseError(err error) bool {
	msg := err.Error()
	return strings.HasPrefix(msg, "unknown command")
}
