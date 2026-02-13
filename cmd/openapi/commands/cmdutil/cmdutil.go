// Package cmdutil provides shared CLI utilities for all command groups.
package cmdutil

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// StdinIndicator is the conventional Unix indicator to read from stdin.
const StdinIndicator = "-"

// IsStdin returns true if the given path indicates stdin should be used.
func IsStdin(path string) bool {
	return path == StdinIndicator
}

// StdinIsPiped returns true when stdin is connected to a pipe (not a terminal),
// meaning data is being piped in from another command or a file redirect.
func StdinIsPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// InputFileFromArgs returns the input file from args, or "-" if stdin should
// be used. It extracts the first positional arg or detects piped stdin.
func InputFileFromArgs(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return StdinIndicator
}

// StdinOrFileArgs returns a cobra arg validator that accepts minArgs..maxArgs
// when a file is given, but also allows zero args when stdin is piped.
func StdinOrFileArgs(minArgs, maxArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if StdinIsPiped() {
				return nil
			}
			return fmt.Errorf("requires at least %d arg(s), or pipe data to stdin", minArgs)
		}
		if len(args) < minArgs {
			return fmt.Errorf("requires at least %d arg(s), only received %d", minArgs, len(args))
		}
		if maxArgs >= 0 && len(args) > maxArgs {
			return fmt.Errorf("accepts at most %d arg(s), received %d", maxArgs, len(args))
		}
		return nil
	}
}

// Dief prints a formatted message to stderr and exits with code 1.
func Dief(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
	os.Exit(1)
}

// Die prints an error to stderr and exits with code 1.
func Die(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
