package vm

import (
	"fmt"
	"io/fs"
	"os"
)

// DirCodeProvider is a filesystem-based code provider that loads Lua files
// relative to a root directory. It uses os.DirFS for path containment.
type DirCodeProvider struct {
	fs   fs.FS
	caps LuaLoaderCaps
}

// NewDirCodeProvider creates a code provider rooted at the given directory.
func NewDirCodeProvider(root string, caps LuaLoaderCaps) *DirCodeProvider {
	return &DirCodeProvider{
		fs:   os.DirFS(root),
		caps: caps,
	}
}

// LoadChunk reads a Lua source file from the jailed directory.
func (p *DirCodeProvider) LoadChunk(name string, caller *LuaCallerContext) ([]byte, string, error) {
	data, err := fs.ReadFile(p.fs, name)
	if err != nil {
		return nil, "", fmt.Errorf("cannot open %s: %v", name, err)
	}
	return data, "@" + name, nil
}

// Capabilities returns the configured loader capabilities.
func (p *DirCodeProvider) Capabilities() LuaLoaderCaps {
	return p.caps
}
