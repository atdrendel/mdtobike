package cmd

import "errors"

// ErrCancelled is returned when the user cancels an operation.
// This is not an error condition — the user made a valid choice.
var ErrCancelled = errors.New("cancelled")

// ErrSilent is returned when a command fails but has already printed
// appropriate error messages. The CLI should exit with code 1 but not
// print an additional "Error:" message.
var ErrSilent = errors.New("silent error")
