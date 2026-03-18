package vm

import (
	"os"
	"os/exec"
	"syscall"
)

// LuaExecProvider is a capability interface for sandboxed command execution.
// When provided to a VM, os.execute becomes available.
type LuaExecProvider interface {
	// Execute runs a shell command and returns the result.
	// If command is empty, returns (true, "exit", 0) to indicate a shell is available.
	// Otherwise returns (ok, exitType, exitCode) where:
	//   - ok is true if the command exited with code 0
	//   - exitType is "exit" for normal termination or "signal" for signal termination
	//   - exitCode is the exit code or signal number
	Execute(command string) (ok bool, exitType string, exitCode int)
}

// DefaultExecProvider implements LuaExecProvider using os/exec.
// Commands are run via "sh -c <command>".
type DefaultExecProvider struct{}

// NewDefaultExecProvider creates a new DefaultExecProvider.
func NewDefaultExecProvider() *DefaultExecProvider {
	return &DefaultExecProvider{}
}

// Execute runs a command via sh -c.
func (p *DefaultExecProvider) Execute(command string) (bool, string, int) {
	if command == "" {
		return true, "exit", 0
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					return false, "signal", int(status.Signal())
				}
				return false, "exit", status.ExitStatus()
			}
			return false, "exit", exitErr.ExitCode()
		}
		// Command not found or other error
		return false, "exit", 127
	}
	return true, "exit", 0
}
