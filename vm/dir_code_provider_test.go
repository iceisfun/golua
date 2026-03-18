package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirCodeProvider_DotPathComponent(t *testing.T) {
	// Create a temp directory with a file
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "test.lua"), []byte("return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := NewDirCodeProvider(dir, LuaLoaderCaps{})

	// "sub/test.lua" should work
	_, _, err := provider.LoadChunk("sub/test.lua", nil)
	if err != nil {
		t.Fatalf("LoadChunk sub/test.lua failed: %v", err)
	}

	// "sub/./test.lua" should also work (os.DirFS rejects "." components
	// unless the path is cleaned first)
	_, _, err = provider.LoadChunk("sub/./test.lua", nil)
	if err != nil {
		t.Fatalf("LoadChunk sub/./test.lua failed: %v", err)
	}

	// "./sub/test.lua" should also work
	_, _, err = provider.LoadChunk("./sub/test.lua", nil)
	if err != nil {
		t.Fatalf("LoadChunk ./sub/test.lua failed: %v", err)
	}
}
