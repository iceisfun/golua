package vm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
			return nil, "", fmt.Errorf("cannot open %s: %v", name, unwrapFileError(err))
		}
		if strings.HasPrefix(absName, p.root) || strings.HasPrefix(absName, os.TempDir()) {
			data, err := os.ReadFile(absName)
			if err != nil {
				return nil, "", fmt.Errorf("cannot open %s: %v", name, unwrapFileError(err))
			}
			return data, "@" + name, nil
		}
		return nil, "", fmt.Errorf("cannot open %s: access denied", name)
	}

	data, err := fs.ReadFile(p.fs, name)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open %s: %v", name, unwrapFileError(err))
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
		return pe.Err
	}
	return err
}
