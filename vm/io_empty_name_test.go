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

// A final "." or ".." component must not be cleaned away: "." "./" and "x/.."
// all clean to the jail root, and os.Remove would delete the root once it was
// empty ("a/b/.." would likewise delete "a"). The kernel never lets rmdir or
// rename act through such a name; the provider must answer the same way.
func TestRemoveRenameDotNamesKeepDirectories(t *testing.T) {
	root := t.TempDir()
	p := vm.NewFullIoProvider(root)
	ctx := context.Background()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}

	removes := []struct {
		name string
		want syscall.Errno
	}{
		{".", syscall.EINVAL},
		{"./", syscall.EINVAL},
		{".//", syscall.EINVAL},
		{"a/.", syscall.EINVAL},
		{"a/./", syscall.EINVAL},
		{"a/..", syscall.ENOTEMPTY},
		{"a/b/..", syscall.ENOTEMPTY},
		{"..", syscall.ENOTEMPTY},
		{"missing/..", syscall.ENOENT},
	}
	renames := []string{".", "./", "a/.", "a/..", "a/b/..", ".."}

	check := func() {
		t.Helper()
		for _, tc := range removes {
			if err := p.Remove(ctx, tc.name); !errors.Is(err, tc.want) {
				t.Errorf("Remove(%q) = %v, want %v", tc.name, err, tc.want)
			}
		}
		for _, name := range renames {
			if err := p.Rename(ctx, name, "zz"); !errors.Is(err, syscall.EBUSY) {
				t.Errorf("Rename(%q, zz) = %v, want EBUSY", name, err)
			}
			if err := p.Rename(ctx, "a", name); !errors.Is(err, syscall.EBUSY) {
				t.Errorf("Rename(a, %q) = %v, want EBUSY", name, err)
			}
		}
		for _, dir := range []string{root, filepath.Join(root, "a"), filepath.Join(root, "a", "b")} {
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("%s was removed: %v", dir, err)
			}
		}
	}
	check()

	// With b gone, a/b/.. would clean to an empty a, and with a gone too, "."
	// would clean to an empty root: the cases that actually deleted things.
	if err := os.Remove(filepath.Join(root, "a", "b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := p.Remove(ctx, "a/b/.."); !errors.Is(err, syscall.ENOTEMPTY) {
		t.Fatalf("Remove(a/b/..) = %v, want ENOTEMPTY", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".", "./", "a/.."} {
		if err := p.Remove(ctx, name); err == nil {
			t.Fatalf("Remove(%q) on an empty root succeeded", name)
		}
		if _, err := os.Stat(root); err != nil {
			t.Fatalf("Remove(%q) removed the root: %v", name, err)
		}
	}

	// A file is not a directory to take "." of.
	if err := os.WriteFile(filepath.Join(root, "f"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := p.Remove(ctx, "f/."); !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("Remove(f/.) = %v, want ENOTDIR", err)
	}
	if err := os.Remove(filepath.Join(root, "f")); err != nil {
		t.Fatal(err)
	}

	// Other spellings of the (now empty) root are refused as well.
	for _, name := range []string{root, "../" + filepath.Base(root)} {
		if err := p.Remove(ctx, name); err == nil {
			t.Fatalf("Remove(%q) on an empty root succeeded", name)
		}
		if err := p.Rename(ctx, name, "zz"); err == nil {
			t.Fatalf("Rename(%q, zz) succeeded", name)
		}
		if _, err := os.Stat(root); err != nil {
			t.Fatalf("Remove(%q) removed the root: %v", name, err)
		}
	}

	// Ordinary names still work.
	if err := os.Mkdir(filepath.Join(root, "d"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := p.Rename(ctx, "d/", "e"); err != nil {
		t.Fatalf("Rename(d/, e) = %v", err)
	}
	if err := p.Remove(ctx, "e/"); err != nil {
		t.Fatalf("Remove(e/) = %v", err)
	}
}
