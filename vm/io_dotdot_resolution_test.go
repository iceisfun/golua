package vm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// The kernel applies "." and ".." to the directory it has reached, after
// following the symlinks before them, and only through components that exist
// and are directories. Cleaning a name lexically first made "missing/../x"
// and "file/../x" name x, so open truncated it and remove/rename destroyed or
// moved it, and "lnk/../x" read x next to the link instead of next to its
// target.
func TestJailResolvesDotDotLikeTheKernel(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a", "b"))
	writeFile(t, filepath.Join(root, "x"), "top")
	writeFile(t, filepath.Join(root, "a", "x"), "inner")
	writeFile(t, filepath.Join(root, "ff"), "file")
	symlink(t, filepath.Join("a", "b"), filepath.Join(root, "lnk"))

	p := vm.NewFullIoProvider(root)
	ctx := context.Background()

	f, err := p.Open(ctx, "lnk/../x", "r")
	if err != nil {
		t.Fatalf("Open(lnk/../x): %v", err)
	}
	got, _ := f.Read(ctx, "l")
	f.Close(ctx)
	if got != "inner" {
		t.Fatalf("Open(lnk/../x) read %q, want the file beside the link target (inner)", got)
	}

	for _, tc := range []struct {
		name string
		want syscall.Errno
	}{
		{"nonexist/../x", syscall.ENOENT},
		{"nonexist/./x", syscall.ENOENT},
		{"ff/../x", syscall.ENOTDIR},
		{"ff/.", syscall.ENOTDIR},
		{"ff/../ff", syscall.ENOTDIR},
	} {
		for _, mode := range []string{"r", "w", "a", "r+"} {
			if f, err := p.Open(ctx, tc.name, mode); !errors.Is(err, tc.want) {
				if err == nil {
					f.Close(ctx)
				}
				t.Errorf("Open(%q, %q) = %v, want %v", tc.name, mode, err, tc.want)
			}
		}
		if err := p.Remove(ctx, tc.name); !errors.Is(err, tc.want) {
			t.Errorf("Remove(%q) = %v, want %v", tc.name, err, tc.want)
		}
		if err := p.Rename(ctx, tc.name, "moved"); !errors.Is(err, tc.want) {
			t.Errorf("Rename(%q, moved) = %v, want %v", tc.name, err, tc.want)
		}
		if err := p.Rename(ctx, "ff", tc.name); !errors.Is(err, tc.want) {
			t.Errorf("Rename(ff, %q) = %v, want %v", tc.name, err, tc.want)
		}
	}
	for name, want := range map[string]string{"x": "top", "ff": "file"} {
		if data, err := os.ReadFile(filepath.Join(root, name)); err != nil || string(data) != want {
			t.Fatalf("%s was touched: data=%q err=%v", name, data, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, "moved")); err == nil {
		t.Fatalf("a failing rename created moved")
	}

	// ".." through directories that exist still works.
	if f, err := p.Open(ctx, "a/b/../../x", "r"); err != nil {
		t.Fatalf("Open(a/b/../../x): %v", err)
	} else {
		f.Close(ctx)
	}
}

// A walk that leaves the jail through ".." and fails out there must not say
// so: whether "../probe/../root/ok.txt" fails tells the script whether probe
// exists beside the root.
func TestJailDoesNotReportFailuresOutsideIt(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	if f, err := p.Open(ctx, "../outdir/../root/ok.txt", "r"); err != nil {
		t.Fatalf("a name that walks back into the jail was refused: %v", err)
	} else {
		f.Close(ctx)
	}
	for _, name := range []string{"../absent/../root/ok.txt", "../SECRET.txt/../root/ok.txt"} {
		f, err := p.Open(ctx, name, "r")
		if err == nil {
			f.Close(ctx)
			t.Fatalf("Open(%q) succeeded through a path the kernel cannot walk", name)
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Errorf("Open(%q) = %v, want access denied", name, err)
		}
	}
}
