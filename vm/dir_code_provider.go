package vm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// DirCodeProvider is a filesystem-based code provider that loads Lua files
// relative to a root directory. It uses os.DirFS for path containment.
// Absolute paths in the OS temp directory are also allowed.
type DirCodeProvider struct {
	fs   fs.FS
	root string
	caps LuaLoaderCaps
}

// NewDirCodeProvider creates a code provider rooted at the given directory.
func NewDirCodeProvider(root string, caps LuaLoaderCaps) *DirCodeProvider {
	absRoot, _ := filepath.Abs(root)
	return &DirCodeProvider{
		fs:   os.DirFS(root),
		root: absRoot,
		caps: caps,
	}
}

// LoadChunk reads a Lua source file from the jailed directory.
// Absolute paths in the OS temp directory or within the root are also allowed.
func (p *DirCodeProvider) LoadChunk(name string, caller *LuaCallerContext) ([]byte, string, error) {
	if filepath.IsAbs(name) {
		// Allow absolute paths within root or temp directory
		absName, err := filepath.Abs(name)
		if err != nil {
			return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
		}
		if strings.HasPrefix(absName, p.root) || strings.HasPrefix(absName, os.TempDir()) {
			data, err := os.ReadFile(absName)
			if err != nil {
				return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
			}
			return data, "@" + name, nil
		}
		return nil, "", fmt.Errorf("cannot open %s: access denied", name)
	}

	data, err := fs.ReadFile(p.fs, name)
	if err != nil {
		return nil, "", fmt.Errorf("cannot %s %s: %v", fileErrorVerb(err), name, unwrapFileError(err))
	}
	return data, "@" + name, nil
}

// Capabilities returns the configured loader capabilities.
func (p *DirCodeProvider) Capabilities() LuaLoaderCaps {
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
