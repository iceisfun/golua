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

// A walk that steps outside the jail through ".." must be refused before it
// looks at anything out there, whatever is there: if "../probe/../root/ok.txt"
// opened when probe is a directory and failed otherwise, the outcome would say
// whether probe exists beside the root. Walking up through the root's own
// ancestors and straight back into it reveals nothing and still works.
func TestJailDoesNotProbeOutsideIt(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	for _, name := range []string{
		"../root/ok.txt",
		"sub/../../root/ok.txt",
		filepath.Join(l.root, "ok.txt"),
		l.root + "/../root/ok.txt",
	} {
		f, err := p.Open(ctx, name, "r")
		if err != nil {
			t.Fatalf("Open(%q) through the root's own ancestors was refused: %v", name, err)
		}
		f.Close(ctx)
	}

	for _, name := range []string{
		"../outdir/../root/ok.txt",     // a directory that exists
		"../absent/../root/ok.txt",     // nothing there
		"../SECRET.txt/../root/ok.txt", // a plain file
		"../outdir/",
		"../absent",
		filepath.Join(l.base, "outdir") + "/../root/ok.txt",
		filepath.Join(l.base, "absent") + "/../root/ok.txt",
		"/nonexistent_golua_probe/.." + l.root + "/ok.txt",
	} {
		for _, mode := range []string{"r", "w"} {
			f, err := p.Open(ctx, name, mode)
			if err == nil {
				f.Close(ctx)
				t.Fatalf("Open(%q, %q) succeeded through a lookup outside the jail", name, mode)
			}
			if !strings.Contains(err.Error(), "access denied") {
				t.Errorf("Open(%q, %q) = %v, want access denied", name, mode, err)
			}
		}
		if err := p.Remove(ctx, name); err == nil || !strings.Contains(err.Error(), "access denied") {
			t.Errorf("Remove(%q) = %v, want access denied", name, err)
		}
	}
	if data, err := os.ReadFile(filepath.Join(l.root, "ok.txt")); err != nil || string(data) != "inside" {
		t.Fatalf("ok.txt was touched: data=%q err=%v", data, err)
	}
}

// A root the host named through a symlink is still reachable by absolute
// names spelled that way: the walk passes the link and its target's ancestors,
// all of which are on the way to the root.
func TestJailAdmitsTheRootsGivenSpelling(t *testing.T) {
	base := t.TempDir()
	mkdir(t, filepath.Join(base, "real", "root"))
	writeFile(t, filepath.Join(base, "real", "root", "ok.txt"), "inside")
	symlink(t, filepath.Join(base, "real"), filepath.Join(base, "alias"))
	given := filepath.Join(base, "alias", "root")

	p := vm.NewFullIoProvider(given)
	ctx := context.Background()
	f, err := p.Open(ctx, filepath.Join(given, "ok.txt"), "r")
	if err != nil {
		t.Fatalf("Open through the given spelling of the root: %v", err)
	}
	f.Close(ctx)
}
