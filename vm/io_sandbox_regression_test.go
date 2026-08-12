package vm_test

// Regression tests for the filesystem sandbox: FullIoProvider's root jail,
// JailedIoProvider's read-only jail and DirCodeProvider's code jail must
// contain every name a script can construct; resolving a name must cost time
// proportional to its length and no more; remove/rename must act on a symlink
// rather than on what it points at; and no read or setvbuf size a script can
// name may reach an allocation the Go runtime kills the process over.
//
// This is an external test package so the escape vectors can be driven through
// real Lua (io.open, io.lines, os.remove/rename, dofile/require), which needs
// the stdlib that imports vm.

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// jailLayout is a filesystem laid out for escape attempts: a jail root with a
// secret next to it, a sibling directory whose name shares the root's prefix,
// and symlinks inside the root aimed out of it.
type jailLayout struct {
	base    string // parent of the jail
	root    string // the jail
	secret  string // base/SECRET.txt, outside the jail
	sibling string // base/root-evil/e.txt, outside the jail
	tmpFile string // a file in the OS temp directory, outside the jail
}

func newJailLayout(t *testing.T) jailLayout {
	t.Helper()
	base := t.TempDir()
	l := jailLayout{
		base:    base,
		root:    filepath.Join(base, "root"),
		secret:  filepath.Join(base, "SECRET.txt"),
		sibling: filepath.Join(base, "root-evil", "e.txt"),
	}
	mkdir(t, filepath.Join(l.root, "sub"))
	mkdir(t, filepath.Join(base, "root-evil"))
	mkdir(t, filepath.Join(base, "outdir"))
	writeFile(t, l.secret, "SECRET")
	writeFile(t, l.sibling, "EVIL")
	writeFile(t, filepath.Join(l.root, "ok.txt"), "inside")
	writeFile(t, filepath.Join(l.root, "sub", "n.txt"), "nested")

	symlink(t, l.secret, filepath.Join(l.root, "link.txt"))                        // link out of the jail
	symlink(t, filepath.Join(base, "root-evil"), filepath.Join(l.root, "evildir")) // linked parent directory
	symlink(t, filepath.Join(base, "NEW.txt"), filepath.Join(l.root, "dangling.txt"))
	symlink(t, filepath.Join(l.root, "ok.txt"), filepath.Join(l.root, "inlink.txt")) // link that stays inside
	symlink(t, filepath.Join(l.root, "sub"), filepath.Join(l.root, "subalias"))      // linked directory that stays inside

	tmp, err := os.CreateTemp("", "golua_sandbox_")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	tmp.WriteString("print('PAYLOAD')\n")
	tmp.Close()
	t.Cleanup(func() { os.Remove(tmp.Name()) })
	l.tmpFile = tmp.Name()
	return l
}

// escapeNames returns every name that must be refused for l's jail.
func (l jailLayout) escapeNames() map[string]string {
	return map[string]string{
		"relative dotdot":     "../SECRET.txt",
		"nested dotdot":       "sub/../../SECRET.txt",
		"deep dotdot":         "sub/../../../../../../etc/passwd",
		"absolute outside":    l.secret,
		"absolute sibling":    l.sibling, // shares the root's string prefix
		"relative sibling":    "../root-evil/e.txt",
		"symlinked file":      "link.txt",
		"symlinked directory": "evildir/e.txt",
		"dangling symlink":    "dangling.txt",
		"temp directory":      l.tmpFile,
	}
}

// escapeNamesOutsideParents is the subset of escapeNames whose PARENT
// directory already lies outside the jail. Those are the names remove/rename
// must refuse: a name that is itself a link inside the root ("link.txt") does
// name an entry inside the jail, and unlinking that entry is legal — see
// TestFullIoProviderJail_RemoveAndRenameActOnTheLink.
func (l jailLayout) escapeNamesOutsideParents() map[string]string {
	names := l.escapeNames()
	delete(names, "symlinked file")
	delete(names, "dangling symlink")
	return names
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("Symlink %s -> %s: %v", link, target, err)
	}
}

// runJailedLua runs source in a VM whose io and code providers are both jailed
// to root, and returns the run error (nil on success).
func runJailedLua(t *testing.T, root, source string) error {
	t.Helper()
	ast, err := parser.Parse("=sandbox", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("=sandbox", ast)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New()
	v.SetIoProvider(vm.NewFullIoProvider(root))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	v.SetCodeProvider(vm.NewDirCodeProvider(root, vm.LuaLoaderCaps{
		AllowDofile:   true,
		AllowLoadfile: true,
	}))
	stdlib.Open(v)
	_, runErr := v.Run(proto)
	return runErr
}

func mustRunJailedLua(t *testing.T, root, source string) {
	t.Helper()
	if err := runJailedLua(t, root, source); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

// --- SBX-1: FullIoProvider root jail ---

func TestFullIoProviderJail_OpenDeniesEscapes(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	for label, name := range l.escapeNames() {
		for _, mode := range []string{"r", "rb", "w", "a", "r+", "w+", "a+"} {
			f, err := p.Open(ctx, name, mode)
			if err == nil {
				f.Close(ctx)
				t.Errorf("%s (%q, mode %q): open succeeded, jail escaped", label, name, mode)
				continue
			}
			if !strings.Contains(err.Error(), "access denied") {
				t.Errorf("%s (%q, mode %q): expected access denied, got %v", label, name, mode, err)
			}
		}
	}

	// A write mode must not have truncated anything outside the jail...
	for _, path := range []string{l.secret, l.sibling, l.tmpFile} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("outside file %s disappeared: %v", path, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("outside file %s was truncated by a jailed write", path)
		}
	}
	// ...nor created one, through the dangling symlink or otherwise.
	if _, err := os.Stat(filepath.Join(l.base, "NEW.txt")); err == nil {
		t.Errorf("a jailed write created a file outside the root via a dangling symlink")
	}
}

func TestFullIoProviderJail_AllowsLegitimateAccess(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	allowed := map[string]string{
		"plain name":        "ok.txt",
		"subdirectory":      "sub/n.txt",
		"inner dotdot":      "sub/../ok.txt",
		"dot prefix":        "./ok.txt",
		"absolute in root":  filepath.Join(l.root, "ok.txt"),
		"symlink to inside": "inlink.txt",
		"linked subdir":     "subalias/n.txt",
	}
	for label, name := range allowed {
		f, err := p.Open(ctx, name, "r")
		if err != nil {
			t.Errorf("%s (%q): expected success, got %v", label, name, err)
			continue
		}
		f.Close(ctx)
	}

	// A root reached through a symlink is still a valid jail: names inside it
	// resolve, names outside it do not.
	symlink(t, l.root, filepath.Join(l.base, "rootlink"))
	linked := vm.NewFullIoProvider(filepath.Join(l.base, "rootlink"))
	if f, err := linked.Open(ctx, "ok.txt", "r"); err != nil {
		t.Errorf("symlinked root: expected in-jail open to succeed, got %v", err)
	} else {
		f.Close(ctx)
	}
	if f, err := linked.Open(ctx, "../SECRET.txt", "r"); err == nil {
		f.Close(ctx)
		t.Errorf("symlinked root: jail escaped")
	}

	// Creating a new file inside the jail still works.
	f, err := p.Open(ctx, "sub/created.txt", "w")
	if err != nil {
		t.Fatalf("create inside jail: %v", err)
	}
	if err := f.Write(ctx, "hello"); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close(ctx)
	if data, err := os.ReadFile(filepath.Join(l.root, "sub", "created.txt")); err != nil || string(data) != "hello" {
		t.Fatalf("created file: data=%q err=%v", data, err)
	}
}

func TestFullIoProviderJail_RemoveAndRenameJailed(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	for label, name := range l.escapeNamesOutsideParents() {
		if err := p.Remove(ctx, name); err == nil {
			t.Errorf("remove %s (%q) succeeded, jail escaped", label, name)
		}
		// Both rename endpoints are jailed: out of the root...
		if err := p.Rename(ctx, "ok.txt", name); err == nil {
			t.Errorf("rename to %s (%q) succeeded, jail escaped", label, name)
		}
		// ...and into it.
		if err := p.Rename(ctx, name, "stolen.txt"); err == nil {
			t.Errorf("rename from %s (%q) succeeded, jail escaped", label, name)
		}
	}
	if data, err := os.ReadFile(l.secret); err != nil || string(data) != "SECRET" {
		t.Fatalf("secret outside the jail was removed or replaced: data=%q err=%v", data, err)
	}
	if _, err := os.Lstat(filepath.Join(l.root, "stolen.txt")); err == nil {
		t.Errorf("rename smuggled an outside file into the jail")
	}
	if data, err := os.ReadFile(l.sibling); err != nil || string(data) != "EVIL" {
		t.Fatalf("file outside the jail was moved away: data=%q err=%v", data, err)
	}

	// In-jail rename and remove still work.
	if err := p.Rename(ctx, "ok.txt", "sub/moved.txt"); err != nil {
		t.Fatalf("in-jail rename: %v", err)
	}
	if err := p.Remove(ctx, "sub/moved.txt"); err != nil {
		t.Fatalf("in-jail remove: %v", err)
	}
}

// TestFullIoProviderJail_RemoveAndRenameActOnTheLink pins the C semantics the
// jail must not change: remove() unlinks a symlink and rename() moves it, both
// leaving the file it points at untouched. Resolving the final component for
// containment (which is right for open, since fopen follows it) would instead
// destroy the target — silently, for a script that only maintains links inside
// its own data root.
func TestFullIoProviderJail_RemoveAndRenameActOnTheLink(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	target := filepath.Join(l.root, "target.txt")
	writeFile(t, target, "TARGET")
	symlink(t, "target.txt", filepath.Join(l.root, "tolink.txt"))

	if err := p.Remove(ctx, "tolink.txt"); err != nil {
		t.Fatalf("remove of an in-jail symlink: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "TARGET" {
		t.Fatalf("os.remove followed the link and destroyed its target: data=%q err=%v", data, err)
	}
	if _, err := os.Lstat(filepath.Join(l.root, "tolink.txt")); err == nil {
		t.Fatalf("the link itself survived the remove")
	}

	symlink(t, "target.txt", filepath.Join(l.root, "tolink2.txt"))
	if err := p.Rename(ctx, "tolink2.txt", "moved.txt"); err != nil {
		t.Fatalf("rename of an in-jail symlink: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "TARGET" {
		t.Fatalf("os.rename followed the link and moved its target: data=%q err=%v", data, err)
	}
	info, err := os.Lstat(filepath.Join(l.root, "moved.txt"))
	if err != nil {
		t.Fatalf("renamed link is missing: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("rename moved the target instead of the link (moved.txt is not a symlink)")
	}

	// The same holds for a link that points OUT of the jail: unlinking it is an
	// operation on an entry inside the root, and the outside file is untouched.
	if err := p.Remove(ctx, "link.txt"); err != nil {
		t.Fatalf("remove of a link pointing outside: %v", err)
	}
	if data, err := os.ReadFile(l.secret); err != nil || string(data) != "SECRET" {
		t.Fatalf("removing a link destroyed the outside file it pointed at: data=%q err=%v", data, err)
	}
	// ...and it is still not readable through the jail.
	if f, err := p.Open(ctx, "evildir/e.txt", "r"); err == nil {
		f.Close(ctx)
		t.Fatalf("a link out of the jail became readable")
	}
}

// TestJailPathResolutionIsLinear covers the DoS the containment check must not
// introduce: resolving a name a script builds for free ("a/" a few thousand
// times) has to cost time proportional to its length. Resolving the whole
// prefix again per component is quadratic, and no VM deadline can interrupt it
// because the work happens inside one native call.
func TestJailPathResolutionIsLinear(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewFullIoProvider(l.root)
	ctx := context.Background()

	for _, n := range []int{1000, 2000} {
		name := strings.Repeat("a/", n) + "x.txt"
		start := time.Now()
		if f, err := p.Open(ctx, name, "r"); err == nil {
			f.Close(ctx)
			t.Fatalf("%d components: expected the open to fail", n)
		}
		// A linear walk needs a few milliseconds for this; the bound is loose
		// enough for a loaded machine and still nowhere near a quadratic one.
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("resolving %d path components took %v; resolution is not linear", n, elapsed)
		}
	}

	// Past the kernel's own PATH_MAX the name is refused without walking it at
	// all, which is what the OS does with such a name anyway.
	huge := strings.Repeat("a/", 200000) + "x.txt"
	start := time.Now()
	if f, err := p.Open(ctx, huge, "r"); err == nil {
		f.Close(ctx)
		t.Fatalf("an over-long name opened something")
	} else if !strings.Contains(strings.ToLower(err.Error()), "too long") {
		t.Fatalf("over-long name: expected a name-too-long error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("refusing a 200000-component name took %v", elapsed)
	}

	// A symlink loop is refused rather than followed forever.
	symlink(t, filepath.Join(l.root, "loopb"), filepath.Join(l.root, "loopa"))
	symlink(t, filepath.Join(l.root, "loopa"), filepath.Join(l.root, "loopb"))
	if f, err := p.Open(ctx, "loopa", "r"); err == nil {
		f.Close(ctx)
		t.Fatalf("a symlink loop resolved to something")
	}
}

// TestFullIoProviderJail_TmpNameUsable covers the whole os.tmpname idiom under
// both jail shapes: a read-only root (a mounted script or asset directory, the
// normal sandboxed deployment) and a writable one. The name must be mintable
// (reference Lua always produces one), openable through the IO jail, and
// loadable through the CODE jail — which is a separate jail that never sees the
// IO provider's paths, so the runtime mints somewhere both recognize.
func TestFullIoProviderJail_TmpNameUsable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}
	l := newJailLayout(t)
	readOnly := filepath.Join(l.base, "ro")
	mkdir(t, readOnly)
	if err := os.Chmod(readOnly, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(readOnly, 0o755) })

	for _, root := range []string{readOnly, l.root} {
		p := vm.NewFullIoProvider(root)
		ctx := context.Background()

		name, err := p.TmpName(ctx)
		if err != nil {
			t.Fatalf("TmpName with root %s: %v", root, err)
		}
		t.Cleanup(func() { os.Remove(name) })
		// The name never lands in the jail root: for a read-only root it could
		// not, and for a writable one that root is the host's data directory,
		// which a script must not be able to litter.
		if strings.HasPrefix(name, root+string(filepath.Separator)) {
			t.Fatalf("TmpName minted %q inside the jail root %q", name, root)
		}

		// The minted name is usable through the jail...
		f, err := p.Open(ctx, name, "w")
		if err != nil {
			t.Fatalf("open tmpname: %v", err)
		}
		f.Write(ctx, "x")
		f.Close(ctx)
		if err := p.Remove(ctx, name); err != nil {
			t.Fatalf("remove tmpname: %v", err)
		}

		// ...and it did not open the OS temp directory to the jail: a file
		// planted there by anything else on the host stays out of reach.
		if f, err := p.Open(ctx, l.tmpFile, "r"); err == nil {
			f.Close(ctx)
			t.Fatalf("root %s: a file in the OS temp directory was reachable, jail escaped", root)
		}

		// The full idiom, through a VM whose io and code jails are both rooted
		// at the same directory: write a chunk to the name, load it back, and
		// move it to a derived name.
		mustRunJailedLua(t, root, `
			local name = os.tmpname()
			local f = assert(io.open(name, "w"))
			f:write("return 'from tmp'\n")
			f:close()
			assert(dofile(name) == "from tmp", "a jailed VM cannot load its own tmpname")
			assert(loadfile(name)() == "from tmp", "loadfile refused a runtime-minted name")
			assert(os.rename(name, name .. ".renamed"))
			assert(os.remove(name .. ".renamed"))
		`)
	}
}

// TestTmpNameLoadableAcrossProviderPairings is the shape the official suite
// runs: an UNJAILED io provider (files.lua needs /dev/null) paired with a jailed
// code provider. The temp name one provider mints has to be loadable by the
// other even though the two share no configuration at all.
func TestTmpNameLoadableAcrossProviderPairings(t *testing.T) {
	l := newJailLayout(t)
	ctx := context.Background()

	io := vm.NewTestIoProvider()
	name, err := io.TmpName(ctx)
	if err != nil {
		t.Fatalf("TmpName: %v", err)
	}
	t.Cleanup(func() { os.Remove(name) })
	writeFile(t, name, "return 'across'")

	code := vm.NewDirCodeProvider(l.root, vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true})
	if _, _, err := code.LoadChunk(ctx, name, nil); err != nil {
		t.Fatalf("a jailed code provider refused a runtime-minted temp name: %v", err)
	}

	// A file the runtime did not mint is still refused, wherever it sits.
	if _, _, err := code.LoadChunk(ctx, l.tmpFile, nil); err == nil {
		t.Fatalf("a chunk planted in the OS temp directory loaded, code jail escaped")
	}
}

// TestTmpNameStaysOutOfTheJailRoot pins the property the project's own test
// suite depends on: it wires FullIoProvider at the working directory, and many
// of its Lua files call os.tmpname. A temp name minted into the root would
// leave scratch files in the user's source tree.
func TestTmpNameStaysOutOfTheJailRoot(t *testing.T) {
	root := t.TempDir()
	mustRunJailedLua(t, root, `
		-- More names than any fixed-size allow-set would hold. Every one of them
		-- must still be usable at the end: bounding such a set by evicting its
		-- oldest entry turns a name the script legitimately holds into an
		-- "access denied" it can neither predict nor recover from, and reference
		-- Lua never invalidates a name it minted.
		local names = {}
		for i = 1, 300 do
			names[i] = os.tmpname()
			local f = assert(io.open(names[i], "w"))
			f:write("return ", i, "\n")
			f:close()
		end
		for i = 1, 300 do
			assert(dofile(names[i]) == i, "minted name " .. i .. " stopped working")
			assert(os.remove(names[i]))
		end
	`)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("os.tmpname left %d file(s) in the jail root: %v", len(entries), entries)
	}
}

// TestJailReportsRealOsErrors covers the reporting side of containment: a
// permission failure on a path that is legitimately inside the jail is an
// operational error with its own errno, not a sandbox violation.
func TestJailReportsRealOsErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: directory permissions are not enforced")
	}
	l := newJailLayout(t)
	priv := filepath.Join(l.root, "priv")
	mkdir(t, priv)
	writeFile(t, filepath.Join(priv, "f.txt"), "secret")
	if err := os.Chmod(priv, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(priv, 0o755) })

	p := vm.NewFullIoProvider(l.root)
	f, err := p.Open(context.Background(), "priv/f.txt", "r")
	if err == nil {
		f.Close(context.Background())
		t.Fatalf("an unreadable directory was traversable")
	}
	if strings.Contains(err.Error(), "access denied") {
		t.Fatalf("a permission error inside the jail was reported as a sandbox violation: %v", err)
	}
	if !os.IsPermission(err) {
		t.Fatalf("expected a permission error, got %v", err)
	}

	mustRunJailedLua(t, l.root, `
		local f, msg, errno = io.open("priv/f.txt", "r")
		assert(f == nil)
		-- Reference Lua reports "priv/f.txt: Permission denied" with errno 13.
		assert(msg:find("Permission denied", 1, true), "unexpected message: " .. tostring(msg))
		assert(errno == 13, "unexpected errno: " .. tostring(errno))
	`)
}

func TestFullIoProviderJail_LuaEscapesDenied(t *testing.T) {
	l := newJailLayout(t)
	mustRunJailedLua(t, l.root, `
		local escapes = {
			"../SECRET.txt",
			"sub/../../SECRET.txt",
			"../root-evil/e.txt",
			"link.txt",
			"evildir/e.txt",
			"dangling.txt",
			"`+filepath.ToSlash(l.secret)+`",
			"`+filepath.ToSlash(l.tmpFile)+`",
		}
		for _, name in ipairs(escapes) do
			for _, mode in ipairs({"r", "w", "a", "r+", "w+", "a+"}) do
				local f, err = io.open(name, mode)
				assert(f == nil, "io.open escaped the jail: " .. name .. " mode " .. mode)
				assert(err:find("access denied", 1, true), "unexpected error: " .. tostring(err))
			end
			local ok = pcall(io.lines, name)
			assert(not ok, "io.lines escaped the jail: " .. name)
		end

		-- Names whose parent directory is already outside cannot be unlinked or
		-- moved either. ("link.txt" and "dangling.txt" are entries INSIDE the
		-- root, so unlinking them is legal and hits the link, not its target.)
		for _, name in ipairs({"../SECRET.txt", "evildir/e.txt", "`+filepath.ToSlash(l.secret)+`"}) do
			assert(os.remove(name) == nil, "os.remove escaped the jail: " .. name)
			assert(os.rename("ok.txt", name) == nil, "os.rename escaped the jail: " .. name)
			assert(os.rename(name, "stolen.txt") == nil, "os.rename escaped the jail: " .. name)
		end

		-- In-jail access is unaffected.
		local f = assert(io.open("ok.txt", "r"))
		assert(f:read("l") == "inside")
		f:close()
		f = assert(io.open("sub/../ok.txt", "r"))
		assert(f:read("l") == "inside")
		f:close()
		for line in io.lines("sub/n.txt") do assert(line == "nested") end
		local tmp = os.tmpname()
		f = assert(io.open(tmp, "w")); f:write("x"); f:close()
		assert(os.remove(tmp))
	`)
	if data, err := os.ReadFile(l.secret); err != nil || string(data) != "SECRET" {
		t.Fatalf("secret outside the jail was touched: data=%q err=%v", data, err)
	}
}

// --- SBX-1: the read-only jail ships as the sandbox for untrusted scripts ---

func TestJailedIoProviderContainsSymlinks(t *testing.T) {
	l := newJailLayout(t)
	p := vm.NewJailedIoProvider(l.root)
	ctx := context.Background()

	for label, name := range l.escapeNames() {
		if f, err := p.Open(ctx, name, "r"); err == nil {
			f.Close(ctx)
			t.Errorf("%s (%q): read-only jail escaped", label, name)
		}
	}
	for _, name := range []string{"ok.txt", "sub/n.txt", "./ok.txt", "inlink.txt", "subalias/n.txt", filepath.Join(l.root, "ok.txt")} {
		f, err := p.Open(ctx, name, "r")
		if err != nil {
			t.Errorf("%q: expected the read-only jail to allow this, got %v", name, err)
			continue
		}
		f.Close(ctx)
	}
}

// --- Opt-in widening: a host that is not sandboxing ---

func TestProvidersAllowRootWidensTheJail(t *testing.T) {
	l := newJailLayout(t)
	ctx := context.Background()
	writeFile(t, filepath.Join(l.base, "wide.lua"), "return 'wide'")

	io := vm.NewFullIoProvider(l.root)
	if f, err := io.Open(ctx, l.secret, "r"); err == nil {
		f.Close(ctx)
		t.Fatalf("the default jail must not reach outside its root")
	}
	io.AllowRoot(l.base)
	f, err := io.Open(ctx, l.secret, "r")
	if err != nil {
		t.Fatalf("after AllowRoot: %v", err)
	}
	f.Close(ctx)

	code := vm.NewDirCodeProvider(l.root, vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true})
	if _, _, err := code.LoadChunk(ctx, filepath.Join(l.base, "wide.lua"), nil); err == nil {
		t.Fatalf("the default code jail must not load chunks outside its root")
	}
	code.AllowRoot(l.base)
	if _, _, err := code.LoadChunk(ctx, filepath.Join(l.base, "wide.lua"), nil); err != nil {
		t.Fatalf("after AllowRoot: %v", err)
	}

	ro := vm.NewJailedIoProvider(l.root)
	if f, err := ro.Open(ctx, l.secret, "r"); err == nil {
		f.Close(ctx)
		t.Fatalf("the default read-only jail must not reach outside its root")
	}
	ro.AllowRoot(l.base)
	if f, err := ro.Open(ctx, l.secret, "r"); err != nil {
		t.Fatalf("after AllowRoot: %v", err)
	} else {
		f.Close(ctx)
	}
}

// --- SBX-2: DirCodeProvider code jail ---

func TestDirCodeProviderJail_LoadChunkDeniesEscapes(t *testing.T) {
	l := newJailLayout(t)
	writeFile(t, filepath.Join(l.base, "attacker.lua"), "return 'pwned'")
	writeFile(t, filepath.Join(l.base, "root-evil", "e.lua"), "return 'pwned'")
	writeFile(t, filepath.Join(l.root, "mod.lua"), "return 'inside'")
	writeFile(t, filepath.Join(l.root, "sub", "mod.lua"), "return 'nested'")
	symlink(t, filepath.Join(l.base, "attacker.lua"), filepath.Join(l.root, "linked.lua"))

	p := vm.NewDirCodeProvider(l.root, vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true})
	ctx := context.Background()

	denied := map[string]string{
		"relative dotdot":    "../attacker.lua",
		"nested dotdot":      "sub/../../attacker.lua",
		"absolute outside":   filepath.Join(l.base, "attacker.lua"),
		"absolute sibling":   filepath.Join(l.base, "root-evil", "e.lua"),
		"relative sibling":   "../root-evil/e.lua",
		"symlink out":        "linked.lua",
		"symlinked dir":      "evildir/e.lua",
		"temp directory":     l.tmpFile,
		"absolute elsewhere": "/etc/passwd",
	}
	for label, name := range denied {
		if _, _, err := p.LoadChunk(ctx, name, nil); err == nil {
			t.Errorf("%s (%q): chunk loaded, code jail escaped", label, name)
		}
	}

	allowed := []string{"mod.lua", "./mod.lua", "sub/mod.lua", "sub/./mod.lua", "sub/../mod.lua", filepath.Join(l.root, "mod.lua")}
	for _, name := range allowed {
		if _, _, err := p.LoadChunk(ctx, name, nil); err != nil {
			t.Errorf("%q: expected load to succeed, got %v", name, err)
		}
	}

	// A missing file still reports the OS error, not "access denied".
	if _, _, err := p.LoadChunk(ctx, "missing.lua", nil); err == nil {
		t.Errorf("missing.lua: expected an error")
	} else if !strings.Contains(err.Error(), "No such file") {
		t.Errorf("missing.lua: expected a not-found error, got %v", err)
	}
}

func TestDirCodeProviderJail_LuaLoadersDenied(t *testing.T) {
	l := newJailLayout(t)
	writeFile(t, filepath.Join(l.base, "attacker.lua"), "_G.PWNED = true; return true")
	tmpDir := filepath.Dir(l.tmpFile)
	writeFile(t, filepath.Join(l.root, "mod.lua"), "return 'inside'")

	mustRunJailedLua(t, l.root, `
		local outside = "`+filepath.ToSlash(filepath.Join(l.base, "attacker.lua"))+`"
		for _, name in ipairs({"../attacker.lua", outside, "`+filepath.ToSlash(l.tmpFile)+`"}) do
			local chunk, err = loadfile(name)
			assert(chunk == nil, "loadfile escaped the code jail: " .. name)
			assert(err:find("access denied", 1, true), "unexpected error: " .. tostring(err))
			assert(not pcall(dofile, name), "dofile escaped the code jail: " .. name)
		end
		assert(_G.PWNED == nil, "outside chunk executed")

		-- package.path pointing outside the root must not make it loadable.
		package.path = "`+filepath.ToSlash(tmpDir)+`/?.lua;`+filepath.ToSlash(l.base)+`/?.lua"
		assert(not pcall(require, "attacker"), "require escaped the code jail")
		assert(_G.PWNED == nil, "outside chunk executed via require")

		-- In-jail loading still works.
		package.path = "./?.lua"
		assert(dofile("mod.lua") == "inside")
		assert(require("mod") == "inside")
	`)
}

// --- MEM-1(a): setvbuf size ---

func TestFileSetVBufHugeSizeMatchesReference(t *testing.T) {
	l := newJailLayout(t)
	mustRunJailedLua(t, l.root, `
		local f = assert(io.open("vbuf.txt", "w"))
		-- glibc ignores an outsized buffer hint, so reference Lua answers true
		-- for every size up to math.maxinteger. golua must answer the same
		-- while never handing the number to make([]byte, size), which the Go
		-- runtime aborts the process over.
		for _, mode in ipairs({"no", "full", "line"}) do
			for _, size in ipairs({0, 1, 4096, 1 << 30, (1 << 30) + 1, 1 << 31, 1 << 40, math.maxinteger}) do
				local ok, res = pcall(f.setvbuf, f, mode, size)
				assert(ok, "setvbuf(" .. mode .. ", " .. size .. ") raised: " .. tostring(res))
				assert(res == true, "setvbuf(" .. mode .. ", " .. size .. ") returned " .. tostring(res))
			end
		end
		-- A sane size still works and the file still receives its data.
		assert(f:setvbuf("full", 4096) == true)
		f:write("data")
		f:close()
		local r = assert(io.open("vbuf.txt", "r"))
		assert(r:read("a") == "data")
		r:close()
	`)
}

// --- MEM-1(b): f:read(N) allocation, in every provider ---

func TestFileReadHugeCountBoundedByInput(t *testing.T) {
	l := newJailLayout(t)
	writeFile(t, filepath.Join(l.root, "tiny.txt"), "hi")

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	mustRunJailedLua(t, l.root, `
		local f = assert(io.open("tiny.txt", "r"))
		-- Reference Lua returns the two bytes it has; golua must not size its
		-- buffer by the request (1 GiB is fatal in a memory-capped container).
		assert(f:read(1 << 30) == "hi")
		f:close()

		f = assert(io.open("tiny.txt", "r"))
		local ok, err = pcall(f.read, f, 1 << 60)
		assert(ok == false, "an out-of-range count must fail")
		assert(not err:find("runtime error"), "Go runtime panic leaked: " .. err)
		assert(err:find("not enough memory", 1, true), "unexpected error: " .. err)
		f:close()
	`)
	runtime.ReadMemStats(&after)
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 64<<20 {
		t.Fatalf("f:read(1<<30) on a 2-byte file allocated %d bytes", grew)
	}
}

// TestProviderReadBytesHugeCountBoundedByInput is the Go-level half: every
// in-tree LuaFile must survive a count a script can name, not just the one the
// standalone interpreter happens to wire up.
func TestProviderReadBytesHugeCountBoundedByInput(t *testing.T) {
	l := newJailLayout(t)
	writeFile(t, filepath.Join(l.root, "tiny.txt"), "hi")
	ctx := context.Background()

	providers := map[string]vm.LuaIoProvider{
		"FullIoProvider":   vm.NewFullIoProvider(l.root),
		"JailedIoProvider": vm.NewJailedIoProvider(l.root),
	}
	for name, p := range providers {
		f, err := p.Open(ctx, "tiny.txt", "r")
		if err != nil {
			t.Fatalf("%s: open: %v", name, err)
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		data, err := f.ReadBytes(ctx, 1<<30)
		runtime.ReadMemStats(&after)
		f.Close(ctx)
		if err != nil || data != "hi" {
			t.Errorf("%s: ReadBytes(1<<30) = %q, %v; want \"hi\"", name, data, err)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 16<<20 {
			t.Errorf("%s: ReadBytes(1<<30) on a 2-byte file allocated %d bytes", name, grew)
		}
	}
}

// TestFileReadLargeCountReturnsEverything guards the chunked read against the
// obvious way to get it wrong: a count larger than one chunk must still come
// back whole, and a count past the end must stop at the end.
func TestFileReadLargeCountReturnsEverything(t *testing.T) {
	l := newJailLayout(t)
	const size = 40 << 20 // more than one read chunk
	big := strings.Repeat("0123456789abcdef", size/16)
	writeFile(t, filepath.Join(l.root, "big.bin"), big)

	mustRunJailedLua(t, l.root, `
		local f = assert(io.open("big.bin", "rb"))
		local all = f:read(1 << 30)          -- far more than the file holds
		assert(#all == `+strconv.Itoa(size)+`, "short read: got " .. #all)
		assert(all:sub(1, 16) == "0123456789abcdef")
		assert(all:sub(-16) == "0123456789abcdef")
		assert(f:read(1) == nil, "not at end of file")
		f:close()

		f = assert(io.open("big.bin", "rb"))
		local part = f:read(`+strconv.Itoa(size/2)+`)
		assert(#part == `+strconv.Itoa(size/2)+`, "partial read: got " .. #part)
		local rest = f:read("a")
		assert(#rest == `+strconv.Itoa(size/2)+`, "remainder: got " .. #rest)
		f:close()
	`)
}
