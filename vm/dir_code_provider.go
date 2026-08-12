package vm

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"unicode"
)

// DirCodeProvider is a filesystem-based code provider that loads Lua files
// relative to a root directory. Every name — relative or absolute — must
// resolve, symlinks included, to a location inside that root.
type DirCodeProvider struct {
	jail *pathJail
	caps LuaLoaderCaps
}

// NewDirCodeProvider creates a code provider rooted at the given directory. An
// empty root admits nothing: a code provider that loads from everywhere is
// asked for explicitly, with AllowRoot.
func NewDirCodeProvider(root string, caps LuaLoaderCaps) *DirCodeProvider {
	return &DirCodeProvider{
		jail: newPathJail(root),
		caps: caps,
	}
}

// AllowRoot widens the code jail with another directory, for a host that is not
// sandboxing (a standalone interpreter passing "/" behaves like reference Lua,
// which loads a chunk from anywhere). Relative names keep resolving against the
// original root. Call it before the provider is handed to a running VM.
//
// Note what this gives up: whatever directory is opened becomes a place where
// any process that can write there decides what this VM executes.
func (p *DirCodeProvider) AllowRoot(dir string) { p.jail.allowRoot(dir) }

// LoadChunk reads a Lua source file from the jailed directory.
//
// Containment is the same final-path test the IO jail uses (see pathJail), and
// it is the only thing standing between a sandboxed VM and arbitrary code
// execution: this provider decides what dofile/loadfile/require may run. In
// particular the OS temp directory is not loadable — it is world-writable on a
// typical container, so any other process there could otherwise plant a chunk
// for the VM to execute.
//
// The one thing outside the root that does load is a name this runtime minted
// itself, which lives in the runtime's private temp directory (see
// runtimeTempDir). Writing a chunk to os.tmpname() and loading it back is an
// idiom reference Lua supports and the official suite exercises, and it works
// without the host having to wire the two providers together.
func (p *DirCodeProvider) LoadChunk(ctx context.Context, name string, caller *LuaCallerContext) ([]byte, string, error) {
	// An empty name would resolve to the root directory itself; C's fopen("")
	// fails with ENOENT, which is what Lua reports.
	if name == "" {
		err := &fs.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
		return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
	}

	path, err := p.jail.resolve(name)
	if err != nil {
		if isAccessDenied(err) {
			return nil, "", fmt.Errorf("cannot open %s: access denied", name)
		}
		// A real OS failure (permission, name too long) reads as itself.
		return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
	}
	return data, "@" + name, nil
}

// Capabilities returns the configured loader capabilities.
func (p *DirCodeProvider) Capabilities(ctx context.Context) LuaLoaderCaps {
	return p.caps
}

// unwrapFileError extracts the inner OS error from a *fs.PathError or
// *os.PathError, stripping the Go-specific "open filename:" prefix.
// This produces error messages matching Lua 5.4's format:
// "cannot open file.lua: No such file or directory" instead of
// "cannot open file.lua: open file.lua: no such file or directory".
func unwrapFileError(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return capitalizeError(pe.Err)
	}
	return capitalizeError(err)
}

// fileErrorVerb returns the appropriate verb ("open" or "read") for a file
// error, matching Lua 5.4 which distinguishes between open failures (e.g.
// file not found) and read failures (e.g. path is a directory).
func fileErrorVerb(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) && pe.Op == "read" {
		return "read"
	}
	return "open"
}

// capitalizeError returns an error whose message has its first letter
// capitalized. Go's syscall errors use lowercase ("no such file or directory")
// but C's strerror uses title case ("No such file or directory").
func capitalizeError(err error) error {
	msg := err.Error()
	if len(msg) == 0 {
		return err
	}
	r := []rune(msg)
	if unicode.IsUpper(r[0]) {
		return err
	}
	r[0] = unicode.ToUpper(r[0])
	return errors.New(string(r))
}
