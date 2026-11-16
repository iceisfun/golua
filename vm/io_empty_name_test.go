package vm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// The empty name must not anchor to the jail root: C's remove("") and
// rename("", x) fail with ENOENT, and acting on the root would delete it.
func TestRemoveRenameEmptyNameIsENOENT(t *testing.T) {
	root := t.TempDir()
	p := vm.NewFullIoProvider(root)
	ctx := context.Background()

	if err := p.Remove(ctx, ""); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Remove(\"\") = %v, want ENOENT", err)
	}
	if err := p.Rename(ctx, "", "x"); !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("Rename(\"\", x) = %v, want ENOENT", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := p.Rename(ctx, "a", ""); err == nil {
		t.Fatal("Rename(a, \"\") succeeded")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root was removed: %v", err)
	}
}
