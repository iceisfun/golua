package vm

import (
	"context"
	"fmt"
	"os"
)

// LuaPrintProvider controls where print() and warn() output is routed.
// Implement this interface to intercept, prefix, or redirect Lua output
// to your logging infrastructure.
type LuaPrintProvider interface {
	// Print handles output from Lua's print() function.
	// msg is the tab-joined, newline-free string that print() produces.
	Print(ctx context.Context, msg string)

	// Warn handles output from Lua's warn() function.
	// msg already includes the "Lua warning: " prefix.
	Warn(ctx context.Context, msg string)
}

// DefaultPrintProvider writes print() to stdout and warn() to stderr,
// matching standard Lua behavior.
type DefaultPrintProvider struct{}

// NewDefaultPrintProvider creates a DefaultPrintProvider.
func NewDefaultPrintProvider() *DefaultPrintProvider {
	return &DefaultPrintProvider{}
}

// Print writes msg to stdout with a trailing newline.
func (p *DefaultPrintProvider) Print(ctx context.Context, msg string) {
	fmt.Println(msg)
}

// Warn writes msg to stderr with a trailing newline.
func (p *DefaultPrintProvider) Warn(ctx context.Context, msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
