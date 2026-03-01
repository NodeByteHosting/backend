package cli

import (
	"context"
	"fmt"
	"os"
)

// ExitError wraps an error with an exit code for proper CLI termination.
type ExitError struct {
	Err  error
	Code int
}

// Error implements the error interface.
func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

// Exit logs an error message and exits with the given code.
func Exit(code int, msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(code)
}

// ExitErr logs an error and exits with code 1.
func ExitErr(err error, msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", append(args, err)...)
	os.Exit(1)
}

// GetContext returns a context with cancellation support for CLI operations.
func GetContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
