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

// A trailing separator means "this must be a directory" to the kernel, so a
// name like "file/" must fail with ENOTDIR instead of acting on "file".
func TestTrailingSlashKeepsDirectoryMeaning(t *testing.T) {
	root := t.TempDir()
	p := vm.NewFullIoProvider(root)
	ctx := context.Background()
	file := filepath.Join(root, "plain.txt")
	if err := os.WriteFile(file, []byte("d"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := p.Remove(ctx, "plain.txt/"); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Remove(plain.txt/) = %v, want ENOTDIR", err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("Remove(plain.txt/) deleted the file: %v", err)
	}
	if err := p.Rename(ctx, "plain.txt", "moved.txt/"); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Rename(plain.txt, moved.txt/) = %v, want ENOTDIR", err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("Rename to a slashed name moved the file: %v", err)
	}
	if _, err := p.Open(ctx, "plain.txt/", "r"); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Open(plain.txt/) = %v, want ENOTDIR", err)
	}
	if _, err := p.Open(ctx, "nodir/", "w"); !errors.Is(err, syscall.EISDIR) {
		t.Fatalf("Open(nodir/, w) = %v, want EISDIR", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "nodir")); err == nil {
		t.Fatal("Open(nodir/, w) created a regular file")
	}
}

// C's rename() follows rename(2): a directory replaces an empty directory,
// and the other combinations fail with EISDIR / ENOTEMPTY. Go's os.Rename
// reports EEXIST for all of them.
func TestRenameOntoExistingDirectory(t *testing.T) {
	root := t.TempDir()
	p := vm.NewFullIoProvider(root)
	ctx := context.Background()
	for _, d := range []string{"src", "empty", "full"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"src/inner", "full/x", "file"} {
		if err := os.WriteFile(filepath.Join(root, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := p.Rename(ctx, "file", "full"); !errors.Is(err, syscall.EISDIR) {
		t.Fatalf("Rename(file, full) = %v, want EISDIR", err)
	}
	if err := p.Rename(ctx, "src", "full"); !errors.Is(err, syscall.ENOTEMPTY) {
		t.Fatalf("Rename(src, full) = %v, want ENOTEMPTY", err)
	}
	if err := p.Rename(ctx, "src", "empty"); err != nil {
		t.Fatalf("Rename(src, empty) = %v, want success", err)
	}
	if _, err := os.Stat(filepath.Join(root, "empty", "inner")); err != nil {
		t.Fatalf("directory contents did not move: %v", err)
	}
}
