package vm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// FullIoProvider is a full-capability IO provider that allows both reading
// and writing files. It jails file access to a root directory for security.
//
// One provider may be given to several VMs, including VMs running on different
// goroutines; a provider is safe for that. Note that the VMs then share the
// three standard streams the provider exposes, so two VMs reading stdin draw
// from one stream and each line goes to whichever asked for it first.
type FullIoProvider struct {
	jail   *pathJail
	noJail bool // when true, path-jailing is disabled (test-only, unsandboxed)
	stdin  *stdFile
	stdout *stdFile
	stderr *stdFile
}

// NewFullIoProvider creates a new FullIoProvider rooted at the given directory.
// All file operations are restricted to paths within this directory: relative
// and absolute names alike are resolved (following symlinks) and refused unless
// the final location lies inside the root. The OS temp directory is NOT
// reachable from a jailed VM: it is world-writable on a typical host, so any
// other process there could plant a file for the VM to read.
//
// An empty root means the process working directory, which is what it has
// always meant (filepath.Abs("") returns the working directory).
//
// A host that wants a wider reach than one directory — a general-purpose
// interpreter rather than a sandbox — opts into it with AllowRoot.
func NewFullIoProvider(root string) *FullIoProvider {
	if root == "" {
		root = "."
	}
	return &FullIoProvider{
		jail:   newPathJail(root),
		stdin:  &stdFile{file: os.Stdin, name: "stdin", readable: true},
		stdout: &stdFile{file: os.Stdout, name: "stdout", writable: true},
		stderr: &stdFile{file: os.Stderr, name: "stderr", writable: true},
	}
}

// AllowRoot widens the jail with another directory: after this call a name that
// resolves inside dir is reachable as well as one inside the original root.
// Relative names keep resolving against the original root.
//
// It exists for hosts that are not sandboxing — a standalone interpreter passing
// "/" behaves like reference Lua, which has no filesystem restriction at all.
// Call it before the provider is handed to a running VM.
func (p *FullIoProvider) AllowRoot(dir string) { p.jail.allowRoot(dir) }

// NewTestIoProvider creates an UNSANDBOXED, full read/write IO provider with no
// path-jailing: every absolute or relative path the program names is passed
// straight through to the host filesystem (including real devices such as
// /dev/null and /dev/full).
//
// This is intended ONLY for test harnesses that run trusted Lua and need real
// OS device semantics (e.g. the official Lua 5.5 conformance suite's flush
// tests). It is deliberately NOT a safe embedding default: do not use it to run
// untrusted scripts. For sandboxed embedding use NewFullIoProvider (root-jailed)
// or NewJailedIoProvider (read-only) instead.
func NewTestIoProvider() *FullIoProvider {
	return &FullIoProvider{
		jail:   newPathJail(""),
		noJail: true,
		stdin:  &stdFile{file: os.Stdin, name: "stdin", readable: true},
		stdout: &stdFile{file: os.Stdout, name: "stdout", writable: true},
		stderr: &stdFile{file: os.Stderr, name: "stderr", writable: true},
	}
}

// resolvePath resolves a filename to an absolute path within the root,
// following symlinks in every component including the last — which is what
// fopen() does, so it is what Open must do.
func (p *FullIoProvider) resolvePath(name string) (string, error) {
	// Unsandboxed (test-only) mode: pass the name straight through to the host
	// filesystem with no jailing. Relative names resolve against the process cwd.
	if p.noJail {
		return name, nil
	}
	return p.jail.resolve(name)
}

// resolvePathLink is resolvePath for operations that act on the directory entry
// itself rather than on what it points at. C's remove() and rename() unlink and
// move the LINK; dereferencing the last component here would silently destroy
// the target instead, so only the parent chain is resolved.
func (p *FullIoProvider) resolvePathLink(op, name string) (string, error) {
	if p.noJail {
		return name, nil
	}
	return p.jail.resolveLink(op, name)
}

// pathJail confines filesystem names to one or more root directories. Both the
// IO provider and the code provider use it, so a name that io.open refuses is
// refused by dofile as well.
//
// Containment is decided on the FINAL resolved path, because every cheaper test
// is escapable: a relative name is not safe just because it is relative
// ("../x" climbs straight out), a string prefix is not a path boundary (root
// "/srv/scripts" would also grant "/srv/scripts-evil/x"), and an unresolved
// symlink inside the root is a door to anywhere the link points. Names that do
// not exist yet (file creation) resolve through their deepest existing
// ancestor, so a create cannot be aimed through a dangling link either.
//
// A name is never cleaned lexically before it is resolved. The kernel applies
// "." and ".." to the directory it has reached, after following the symlinks
// before them, and only through components that exist and are directories:
// "lnk/../x" is x next to lnk's target, and "missing/../x" or "file/../x" fail.
// Cleaning either one first turned it into plain "x", so a write, remove or
// rename acted on a file the name does not reach.
type pathJail struct {
	mu sync.Mutex
	// roots are absolute and, where the filesystem let them be inspected,
	// symlink-free; roots[0] anchors relative names. A jail with no root at all
	// admits nothing.
	roots []jailRoot
}

// jailRoot is one directory a jail admits, resolved once when it is added.
type jailRoot struct {
	path string
	// resolved records that the walk over path actually succeeded, so path is
	// known to be symlink-free. Only such a root may be used as the starting
	// point of a later resolution (see pathJail.resolveReal); one that could not
	// be inspected — EACCES on a parent, a symlink loop — is still a valid
	// containment boundary, it just cannot be trusted as a shortcut.
	resolved bool
	// given is the absolute spelling the host supplied, before symlinks were
	// resolved. A name may reach the root through it, so its ancestors are
	// on the way to the root just as the resolved path's are.
	given string
}

func newPathJail(root string) *pathJail {
	j := &pathJail{}
	if root != "" {
		j.roots = []jailRoot{resolveJailRoot(root)}
	}
	return j
}

// resolveJailRoot turns a caller-supplied directory into a jail root, resolving
// it once so that per-call resolution does not have to walk it again. A root
// that cannot be resolved keeps its plain absolute form: a root that does not
// exist yet still has a stable name.
func resolveJailRoot(dir string) jailRoot {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return jailRoot{path: dir, given: dir}
	}
	resolved, err := realPath(abs, nil)
	if err != nil {
		return jailRoot{path: abs, given: abs}
	}
	return jailRoot{path: resolved, resolved: true, given: abs}
}

// allowRoot adds another root directory to the jail.
func (j *pathJail) allowRoot(dir string) {
	if dir == "" {
		return
	}
	root := resolveJailRoot(dir)
	j.mu.Lock()
	defer j.mu.Unlock()
	j.roots = append(j.roots, root)
}

// anchor returns the root that relative names resolve against ("" when the jail
// has no root at all, which admits nothing).
func (j *pathJail) anchor() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	if len(j.roots) == 0 {
		return ""
	}
	return j.roots[0].path
}

// contains reports whether an already-resolved absolute path is inside the jail.
func (j *pathJail) contains(path string) bool {
	j.mu.Lock()
	rooted := len(j.roots) > 0
	for _, root := range j.roots {
		if withinDir(root.path, path) {
			j.mu.Unlock()
			return true
		}
	}
	j.mu.Unlock()
	// A rootless jail admits nothing at all, not even a runtime-minted name.
	return rooted && withinRuntimeTempDir(path)
}

// resolveBase returns a directory the resolution of candidate may start from:
// one that is already known to be absolute and symlink-free and that candidate
// lexically sits under. "" means the walk has to start at the filesystem root.
func (j *pathJail) resolveBase(candidate string) string {
	j.mu.Lock()
	for _, root := range j.roots {
		if _, ok := pathBelow(root.path, candidate); ok && root.resolved {
			j.mu.Unlock()
			return root.path
		}
	}
	j.mu.Unlock()
	if tmp := runtimeTempRoot(); tmp != "" {
		if _, ok := pathBelow(tmp, candidate); ok {
			return tmp
		}
	}
	return ""
}

// resolveReal resolves candidate to its real path, starting the walk at
// whichever jail root already covers it. The roots were resolved once when they
// were added, so re-inspecting their components on every io.open, io.lines,
// os.remove, os.rename, loadfile, dofile and require would be one lstat per
// root component per call, on top of the ones the name itself needs.
//
// What that trades away is noticing that the root's OWN prefix was replaced by
// a symlink after the jail was built — which takes write access to a directory
// above the root, i.e. outside the jail entirely. An adversary who has that can
// swap the root between the resolution and the open no matter how often it is
// re-walked, because resolve-then-open is not atomic; re-walking never bought a
// guarantee, only cost.
func (j *pathJail) resolveReal(candidate string) (string, error) {
	if base := j.resolveBase(candidate); base != "" {
		return realPathUnder(base, candidate, j.mayLookUp)
	}
	return realPath(candidate, j.mayLookUp)
}

// candidate turns a caller-supplied name into the absolute path resolution
// starts from. It is not cleaned: see pathJail for why "." and ".." must be
// left for the walk to apply in order.
func (j *pathJail) candidate(name string) (string, error) {
	root := j.anchor()
	if root == "" {
		return "", accessDenied(name)
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	return root + string(filepath.Separator) + name, nil
}

// resolve returns the real path an operation must act on, following symlinks in
// every component, and refuses anything that lands outside the jail.
func (j *pathJail) resolve(name string) (string, error) {
	candidate, err := j.candidate(name)
	if err != nil {
		return "", err
	}
	resolved, err := j.resolveReal(candidate)
	if err != nil {
		return "", j.resolveError(name, candidate, err)
	}
	if !j.contains(resolved) {
		return "", accessDenied(name)
	}
	return keepTrailingSeparator(name, resolved), nil
}

// keepTrailingSeparator re-appends the trailing separator that resolution
// stripped from name. The kernel gives "x/" a meaning "x" does not
// have: it must name a directory (ENOTDIR for a file, EISDIR when open would
// create one), so dropping it let remove/rename/open act on a regular file.
func keepTrailingSeparator(name, resolved string) string {
	if name != "" && os.IsPathSeparator(name[len(name)-1]) &&
		!os.IsPathSeparator(resolved[len(resolved)-1]) {
		return resolved + string(filepath.Separator)
	}
	return resolved
}

// resolveLink is resolve for remove/rename: the parent chain is resolved (that
// is what containment needs) but the final component is left alone, so the
// operation hits the link and not the file it points at. op names the caller
// ("remove" or "rename") in the errors it builds.
//
// Two kinds of name are answered here without touching the filesystem, because
// resolving them would otherwise turn them into a different entry:
//
//   - A name whose final component is "." or ".." ("." "./" "x/.." "e/.") names
//     a directory through a link the kernel never lets remove/rename act on.
//     Cleaning it would yield a real entry — the jail root for "." and "x/..",
//     or the parent for "a/b/.." — and os.Remove would delete it once empty.
//   - A name that resolves to a jail root (or the runtime's temp directory) by
//     any other spelling ("../root", the absolute root): the root is the
//     sandbox boundary, not an entry inside it.
func (j *pathJail) resolveLink(op, name string) (string, error) {
	if name == "" {
		// The empty name would otherwise anchor to the jail root itself, and
		// remove/rename would act on the root directory. C reports ENOENT.
		return "", &fs.PathError{Op: op, Path: name, Err: syscall.ENOENT}
	}
	if err := j.dotLinkError(op, name); err != nil {
		return "", err
	}
	candidate, err := j.candidate(name)
	if err != nil {
		return "", err
	}
	parent, base := splitLastComponent(candidate)
	var resolved string
	if base == "" {
		// A filesystem root has no final component to hold back.
		resolved, err = j.resolve(name)
		if err != nil {
			return "", err
		}
	} else {
		realParent, err := j.resolveReal(parent)
		if err != nil {
			return "", j.resolveError(name, candidate, err)
		}
		resolved = filepath.Join(realParent, base)
		if !j.contains(resolved) {
			return "", accessDenied(name)
		}
	}
	if j.isRoot(resolved) {
		return "", accessDenied(name)
	}
	return keepTrailingSeparator(name, resolved), nil
}

// dotLinkError answers remove/rename of a name whose final component (after
// any trailing separators) is "." or "..", the way the kernel does: the
// directory part must still resolve, and must be a directory, but the
// operation itself always fails — rmdir reports EINVAL for "." and ENOTEMPTY
// for "..", and rename reports EBUSY for either. nil means name is an
// ordinary one.
func (j *pathJail) dotLinkError(op, name string) error {
	trimmed := name
	for len(trimmed) > 1 && os.IsPathSeparator(trimmed[len(trimmed)-1]) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	last := filepath.Base(trimmed)
	if last != "." && last != ".." {
		return nil
	}
	dir := trimmed[:len(trimmed)-len(last)]
	if dir == "" {
		dir = "."
	}
	resolvedDir, err := j.resolve(dir)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			return &fs.PathError{Op: op, Path: name, Err: pe.Err}
		}
		return err
	}
	if fi, err := os.Stat(resolvedDir); err != nil {
		return &fs.PathError{Op: op, Path: name, Err: underlyingErrno(err)}
	} else if !fi.IsDir() {
		return &fs.PathError{Op: op, Path: name, Err: syscall.ENOTDIR}
	}
	errno := syscall.EBUSY
	if op == "remove" {
		errno = syscall.EINVAL
		if last == ".." {
			errno = syscall.ENOTEMPTY
		}
	}
	return &fs.PathError{Op: op, Path: name, Err: errno}
}

// splitLastComponent splits an absolute path into its directory part and its
// final component, ignoring trailing separators, without cleaning either: the
// directory part still has to be resolved component by component. The final
// component is "" for a bare filesystem root.
func splitLastComponent(path string) (dir, last string) {
	trimmed := path
	for len(trimmed) > 1 && os.IsPathSeparator(trimmed[len(trimmed)-1]) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	i := len(trimmed) - 1
	for i >= 0 && !os.IsPathSeparator(trimmed[i]) {
		i--
	}
	if i < 0 {
		return ".", trimmed
	}
	dir = trimmed[:i]
	if dir == "" || dir == filepath.VolumeName(trimmed) {
		dir = trimmed[:i+1]
	}
	return dir, trimmed[i+1:]
}

// underlyingErrno strips the PathError wrapper so the caller's own name can be
// put around the cause.
func underlyingErrno(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// isRoot reports whether an already-resolved path is one of the jail's roots or
// the runtime's temp directory itself.
func (j *pathJail) isRoot(path string) bool {
	j.mu.Lock()
	for _, root := range j.roots {
		if root.path == path {
			j.mu.Unlock()
			return true
		}
	}
	j.mu.Unlock()
	tmp := runtimeTempRoot()
	return tmp != "" && tmp == path
}

// resolveError decides what a resolution failure looks like to the caller. A
// failure that is not about containment — EACCES on a directory the script may
// legitimately name, ENAMETOOLONG, ELOOP — is an operational error and is
// reported as the OS reports it, so a permission problem inside the jail does
// not masquerade as a sandbox violation with the wrong errno. That only applies
// to a name that is lexically inside the jail: for anything else the answer must
// not depend on the state of the filesystem outside it. The same holds for a
// name that is lexically inside but whose walk steps outside: the walk is
// refused before it looks anything up out there (see mayLookUp), so
// "../d/../root/x" is denied whether or not d exists.
func (j *pathJail) resolveError(name, candidate string, err error) error {
	if !j.contains(candidate) {
		return accessDenied(name)
	}
	if errors.Is(err, errLookupOutsideJail) {
		return accessDenied(name)
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		if !j.mayLookUp(pe.Path) {
			return accessDenied(name)
		}
		// Report the name the caller gave rather than the internal path the
		// walk failed on.
		return &fs.PathError{Op: pe.Op, Path: name, Err: pe.Err}
	}
	return err
}

// mayLookUp reports whether a resolution walk may inspect path: path is inside
// the jail, or on the way to one of its roots (a root itself or an ancestor of
// one), which a walk from the filesystem root has to pass through. Anything
// else is refused before it is looked up, whatever is there. Merely hiding the
// failures out there is not enough: if "../d/../root/x" opened whenever d is a
// searchable directory and was denied otherwise, the outcome would still say
// whether d exists beside the root.
func (j *pathJail) mayLookUp(path string) bool {
	if j.contains(path) {
		return true
	}
	j.mu.Lock()
	rooted := len(j.roots) > 0
	for _, root := range j.roots {
		if withinDir(path, root.path) || (root.given != "" && withinDir(path, root.given)) {
			j.mu.Unlock()
			return true
		}
	}
	j.mu.Unlock()
	tmp := runtimeTempRoot()
	return rooted && tmp != "" && withinDir(path, tmp)
}

// errLookupOutsideJail is returned by a guarded walk that would have had to
// inspect a path the jail does not let it look at.
var errLookupOutsideJail = errors.New("path lookup outside the jail")

// accessDeniedError marks a refusal by the jail, as opposed to a failure of the
// operation itself. Callers that phrase their own messages (the code provider)
// need to tell the two apart without matching on text.
type accessDeniedError struct{ name string }

func (e *accessDeniedError) Error() string { return "access denied: " + e.name }

func accessDenied(name string) error { return &accessDeniedError{name: name} }

func isAccessDenied(err error) bool {
	var denied *accessDeniedError
	return errors.As(err, &denied)
}

const (
	// maxSymlinkHops bounds link following, as the kernel's own ELOOP limit does.
	maxSymlinkHops = 40
	// maxPathLen mirrors Linux PATH_MAX. The kernel refuses a longer name with
	// ENAMETOOLONG without looking at it, so refusing it here costs a caller
	// nothing and stops a script from turning free name construction ("a/" a
	// hundred thousand times) into resolution work the VM cannot interrupt.
	maxPathLen = 4096
	// maxPathSteps bounds the components one resolution may visit. Only symlink
	// expansion can push the count past maxPathLen/2, and only up to the hop
	// limit, so no legitimate name comes close.
	maxPathSteps = 16384
)

// realPath returns path — which must be absolute — with every symlink and
// every "." and ".." resolved in order, as the kernel resolves them. Unlike
// filepath.EvalSymlinks it tolerates a final component that does not exist
// yet: it is kept as written, which is where a create would land. A component
// after a missing one or after a non-directory fails with ENOENT or ENOTDIR,
// as it does in the kernel.
//
// Resolution is a single forward walk that inspects each component once, so the
// cost is linear in the length of the name. Resolving the whole prefix again per
// component (what EvalSymlinks does when called level by level) is quadratic,
// and every jailed entry point would inherit it as an uninterruptible CPU sink.
//
// allow, when not nil, is asked about every path before the walk inspects it;
// a refusal ends the walk with errLookupOutsideJail.
func realPath(path string, allow func(string) bool) (string, error) {
	if err := checkPathLen(path); err != nil {
		return "", err
	}
	sep := string(filepath.Separator)
	volume := filepath.VolumeName(path)
	return walkPath(volume+sep, splitPathComponents(path[len(volume):]), path, allow)
}

// checkPathLen refuses an over-long name before anything splits or walks it.
func checkPathLen(path string) error {
	if len(path) > maxPathLen {
		return &fs.PathError{Op: "open", Path: path, Err: syscall.ENAMETOOLONG}
	}
	return nil
}

// realPathUnder is realPath for a path already known to sit under base — an
// absolute directory some jail resolved once and stored symlink-free. Only the
// components below base are inspected: walking the root prefix again on every
// jailed call is one lstat per root component for an answer the jail already
// has. A symlink below base is still followed, and an absolute link target
// restarts the walk at the filesystem root, so nothing is assumed about where
// the name ends up — only about where it starts.
func realPathUnder(base, path string, allow func(string) bool) (string, error) {
	if err := checkPathLen(path); err != nil {
		return "", err
	}
	rest, ok := pathBelow(base, path)
	if !ok {
		// Not spelled under base after all; resolve it the long way.
		return realPath(path, allow)
	}
	return walkPath(base, splitPathComponents(rest), path, allow)
}

// pathBelow returns what follows dir in path when path literally starts with
// dir as a whole number of components. It does not clean path: a walk may
// start at dir and apply the rest, ".." included, in order.
func pathBelow(dir, path string) (string, bool) {
	if path == dir {
		return "", true
	}
	prefix := dir
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	return path[len(prefix):], true
}

// walkPath resolves pending, the components below base, against the filesystem.
// base must be absolute and symlink-free; full is the whole name the caller
// asked about and is used only for error reporting. allow, when not nil, must
// approve each path before it is inspected.
func walkPath(base string, pending []string, full string, allow func(string) bool) (string, error) {
	sep := string(filepath.Separator)
	resolved := base

	hops, steps := 0, 0
	for len(pending) > 0 {
		if steps++; steps > maxPathSteps {
			return "", &fs.PathError{Op: "open", Path: full, Err: syscall.ENAMETOOLONG}
		}
		name := pending[0]
		pending = pending[1:]
		switch name {
		case ".":
			continue
		case "..":
			// Resolved so far is a symlink-free directory, so climbing
			// lexically here is exactly what the kernel does with "..".
			resolved = filepath.Dir(resolved)
			continue
		}

		next := resolved
		if !strings.HasSuffix(next, sep) {
			next += sep
		}
		next += name
		if allow != nil && !allow(next) {
			return "", errLookupOutsideJail
		}

		// Lstat rather than Readlink to decide: "this is not a symlink" is a
		// portable answer from Lstat, while Readlink reports it with a
		// different error on every platform.
		info, err := os.Lstat(next)
		if err != nil {
			// ENOENT and ENOTDIR mean the name does not exist yet. As the
			// final component that is a create target; with more to come the
			// kernel would look them up inside it and fail, "." and ".."
			// included. Anything else — EACCES, ELOOP, ENAMETOOLONG — leaves
			// containment undecidable, so it is surfaced rather than guessed at.
			if !isMissingPathErr(err) {
				return "", err
			}
			if len(pending) > 0 {
				return "", &fs.PathError{Op: "open", Path: next, Err: underlyingErrno(err)}
			}
			resolved = next
			continue
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			if len(pending) > 0 && !info.IsDir() {
				return "", &fs.PathError{Op: "open", Path: next, Err: syscall.ENOTDIR}
			}
			resolved = next
			continue
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", err
		}
		if hops++; hops > maxSymlinkHops {
			return "", &fs.PathError{Op: "open", Path: full, Err: syscall.ELOOP}
		}
		if len(target) > maxPathLen {
			return "", &fs.PathError{Op: "open", Path: full, Err: syscall.ENAMETOOLONG}
		}
		if targetVol := filepath.VolumeName(target); filepath.IsAbs(target) {
			resolved = targetVol + sep
			target = target[len(targetVol):]
		}
		// A relative target continues from the link's own directory, which
		// `resolved` still is: only `next` had the link's name appended.
		pending = append(splitPathComponents(target), pending...)
	}
	return resolved, nil
}

// isMissingPathErr reports whether a lookup failure means "nothing is there"
// rather than "this path cannot be inspected".
func isMissingPathErr(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// splitPathComponents splits a path into its non-empty components. Empty
// entries (leading, trailing and doubled separators) carry no meaning and are
// dropped, which keeps the walk free of special cases.
func splitPathComponents(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == rune(filepath.Separator)
	})
}

// withinDir reports whether path is dir itself or lies beneath it. The
// comparison is component-wise: a string prefix would also accept a sibling
// directory whose name merely starts with dir's.
func withinDir(dir, path string) bool {
	if path == dir {
		return true
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Open opens a file within the provider's root directory.
func (p *FullIoProvider) Open(ctx context.Context, name string, mode string) (LuaFile, error) {
	// Reject empty filenames — Go's os.Open("") opens cwd as a directory,
	// but C's fopen("", ...) returns ENOENT which is what Lua expects.
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	path, err := p.resolvePath(name)
	if err != nil {
		return nil, err
	}

	var flag int
	var readable, writable, seekEnd bool
	switch strings.TrimRight(mode, "b") {
	case "r":
		flag = os.O_RDONLY
		readable = true
	case "w":
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		writable = true
	case "a":
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		writable = true
		// POSIX/glibc position a write-only append stream's indicator at EOF,
		// so seek("cur") reports the end offset (reference Lua parity). Note
		// "a+" deliberately keeps the read position at the start.
		seekEnd = true
	case "r+":
		flag = os.O_RDWR
		readable = true
		writable = true
	case "w+":
		flag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
		readable = true
		writable = true
	case "a+":
		flag = os.O_RDWR | os.O_CREATE | os.O_APPEND
		readable = true
		writable = true
	default:
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	f, err := os.OpenFile(path, flag, 0666)
	if err != nil {
		return nil, err
	}
	if seekEnd {
		// Best-effort: O_APPEND already forces writes to EOF regardless of the
		// position, this only makes seek("cur") report the EOF offset.
		_, _ = f.Seek(0, io.SeekEnd)
	}

	return &fullFile{
		file:     f,
		reader:   bufio.NewReader(f),
		writer:   bufio.NewWriter(f),
		bufMode:  "full",
		readable: readable,
		writable: writable,
	}, nil
}

// Capabilities returns caps with full read/write access.
func (p *FullIoProvider) Capabilities(ctx context.Context) LuaIoCaps {
	return LuaIoCaps{
		AllowRead:  true,
		AllowWrite: true,
	}
}

// Stdin returns the standard input file handle.
func (p *FullIoProvider) Stdin(ctx context.Context) LuaFile { return p.stdin }

// Stdout returns the standard output file handle.
func (p *FullIoProvider) Stdout(ctx context.Context) LuaFile { return p.stdout }

// Stderr returns the standard error file handle.
func (p *FullIoProvider) Stderr(ctx context.Context) LuaFile { return p.stderr }

// TmpName creates a temporary file name. It creates a temp file, closes it,
// removes it, and returns the path -- matching Lua 5.4 os.tmpname behavior.
//
// The name is minted in the runtime's own temp directory (see
// runtimeTempDir), which every jail in this process admits. That is what makes
// the name usable for what a script actually does with it — write a chunk to
// it and load it back — because opening it and loading it are two different
// providers holding two different jails, and the name has to satisfy both.
// Minting there also keeps temp files out of the jail root, which for the
// common read-only-root deployment could not hold them anyway and for a
// writable one is the host's data directory, not a scratch space.
//
// Every name minted stays usable for the life of the process — nothing is
// revoked to bound a set — and the directory outlives the process the way the
// files C's tmpnam leaves behind do, a Go program having no reliable exit hook.
// It is one empty directory per process, against one file per call.
func (p *FullIoProvider) TmpName(ctx context.Context) (string, error) {
	dir, err := runtimeTempDir("")
	if err != nil {
		return "", err
	}
	name, err := mintTmpName(dir)
	if err == nil {
		return name, nil
	}
	// A system temp reaper can delete the directory out from under a
	// long-running process. Rebuild it once rather than failing a call the
	// reference never fails.
	retry, retryErr := runtimeTempDir(dir)
	if retryErr != nil {
		return "", err
	}
	return mintTmpName(retry)
}

// runtimeTempPrefix names the runtime's temp directory recognizably, so an
// operator looking at a host's temp directory can tell what left it there.
const runtimeTempPrefix = "golua-tmp-"

var (
	runtimeTempMu   sync.Mutex
	runtimeTempPath string
)

// runtimeTempDir returns the directory os.tmpname mints names in, creating it
// on first use: one directory inside the OS temp directory, mode 0700, with a
// name nothing outside this process knows in advance. Passing the path of a
// directory that has just failed replaces it (a temp reaper can delete it).
//
// It exists because a minted name is only useful if the script can go on to USE
// it, and using it crosses two providers with two separate jails: the IO
// provider mints the name and opens the file, while the code provider decides
// whether a chunk at that path may run. Neither can learn the other's paths
// unless every host remembers to wire them together, so the runtime mints into
// the one place on the filesystem it created for exactly this purpose and every
// jail recognizes on its own.
//
// The containment this keeps is the one that matters: the OS temp directory
// itself stays unreachable from a jailed VM. It is world-writable on a typical
// host, so anything else there could otherwise plant a file for the VM to read
// or race a name between the mint and the open — a directory this process owns
// at 0700 cannot be read, written, listed or raced from outside the process.
// What a jail admits here is never a pre-existing file and never a name derived
// from script input: only paths this runtime created. Within the process the
// names are mutually reachable, which costs nothing a script does not already
// have — a chunk it wrote itself is one it could equally have run with load().
func runtimeTempDir(retire string) (string, error) {
	runtimeTempMu.Lock()
	defer runtimeTempMu.Unlock()
	if runtimeTempPath != "" && runtimeTempPath != retire {
		return runtimeTempPath, nil
	}
	dir, err := os.MkdirTemp("", runtimeTempPrefix)
	if err != nil {
		return "", err
	}
	runtimeTempPath = resolveJailRoot(dir).path
	return runtimeTempPath, nil
}

// runtimeTempRoot returns the runtime temp directory if one has been created,
// and "" otherwise. A containment check must not create anything, so a process
// that never calls os.tmpname never grows a temp directory.
func runtimeTempRoot() string {
	runtimeTempMu.Lock()
	defer runtimeTempMu.Unlock()
	return runtimeTempPath
}

// withinRuntimeTempDir reports whether an already-resolved path is a name this
// runtime minted (or one a script placed beside it, which is the same thing:
// the directory holds nothing else).
func withinRuntimeTempDir(path string) bool {
	tmp := runtimeTempRoot()
	return tmp != "" && withinDir(tmp, path)
}

// mintTmpName creates a uniquely named file in dir ("" = the OS temp
// directory), then removes it and returns the name, as os.tmpname is specified
// to do. Creating it first is what makes the name unique.
func mintTmpName(dir string) (string, error) {
	f, err := os.CreateTemp(dir, "lua_")
	if err != nil {
		return "", err
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return name, nil
}

// Remove removes a file or empty directory. A symlink is unlinked itself, not
// followed (C's remove() behaves that way, and a script maintaining links
// inside its own data root must not have their targets destroyed).
func (p *FullIoProvider) Remove(ctx context.Context, name string) error {
	path, err := p.resolvePathLink("remove", name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// TmpFile creates and opens a temporary file for read/write.
// The file is automatically removed when closed. The OS temp directory is safe
// here even for a jailed VM: the file is unlinked immediately, so the script
// only ever holds the handle and can never name (or reach) anything else.
func (p *FullIoProvider) TmpFile(ctx context.Context) (LuaFile, error) {
	f, err := os.CreateTemp("", "lua_tmpfile_")
	if err != nil {
		return nil, err
	}
	// Remove the file immediately so it's cleaned up when closed
	os.Remove(f.Name())
	return &fullFile{
		file:     f,
		reader:   bufio.NewReader(f),
		writer:   bufio.NewWriter(f),
		bufMode:  "full",
		readable: true,
		writable: true,
	}, nil
}

// Rename renames (moves) a file within the provider's root directory. Both
// endpoints are jailed: a rename out of the root leaks the file, a rename in
// (from an attacker-planted path) smuggles one back. As with Remove, a symlink
// endpoint is the link itself — C's rename() moves the link, leaving whatever
// it points at where it is.
func (p *FullIoProvider) Rename(ctx context.Context, oldname, newname string) error {
	oldpath, err := p.resolvePathLink("rename", oldname)
	if err != nil {
		return err
	}
	newpath, err := p.resolvePathLink("rename", newname)
	if err != nil {
		return err
	}
	// os.Rename refuses to replace an existing directory (it reports EEXIST
	// before ever calling rename(2)), which C's rename() does not: that
	// replaces an empty directory and answers EISDIR / ENOTEMPTY otherwise.
	// Call the syscall itself so the kernel decides, as it does for C Lua.
	if err := syscall.Rename(oldpath, newpath); err != nil {
		return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: err}
	}
	return nil
}

// fullFile wraps an os.File with buffered reading and writing.
type fullFile struct {
	file     *os.File
	reader   *bufio.Reader
	writer   *bufio.Writer
	closed   bool
	bufMode  string // "no", "full", "line"
	bufSize  int    // buffer size for "full" mode
	readable bool
	writable bool
}

// beforeRead publishes pending writes so a read that follows a write on the
// same stream sees them and starts at the logical position. glibc does this
// implicitly; the separate bufio.Writer would otherwise flush at the wrong
// offset, after the reader's read-ahead. A failed flush fails the read: the
// pending bytes never reached the file, so reading past them would return
// stale data, and bufio keeps the error sticky, so dropping it here would
// only defer it to an unrelated later write or close (glibc reports it
// through the read, via ferror).
func (f *fullFile) beforeRead() error {
	if f.writer != nil && f.writer.Buffered() > 0 {
		return f.writer.Flush()
	}
	return nil
}

// beforeWrite drops the reader's read-ahead and moves the descriptor back to
// the logical position, so a write that follows a read lands where the read
// stopped instead of where the read-ahead did.
func (f *fullFile) beforeWrite() {
	if f.readable && f.reader.Buffered() > 0 {
		if _, err := f.file.Seek(-int64(f.reader.Buffered()), io.SeekCurrent); err == nil {
			f.reader.Reset(f.file)
		}
	}
}

func (f *fullFile) Read(ctx context.Context, format string) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
	}
	if err := f.beforeRead(); err != nil {
		return "", err
	}

	// Normalize format: strip leading * and check first character
	// This matches Lua 5.4 behavior where "all" == "a", "line" == "l", etc.
	cleanFmt := strings.TrimPrefix(format, "*")
	if len(cleanFmt) > 0 {
		switch cleanFmt[0] {
		case 'a':
			data, err := io.ReadAll(f.reader)
			if err != nil {
				return "", err
			}
			return string(data), nil

		case 'l':
			return f.readLine(false)

		case 'L':
			return f.readLine(true)

		case 'n':
			return f.readNumber()
		}
	}

	return "", fmt.Errorf("invalid read format: %s", format)
}

func (f *fullFile) ReadBytes(ctx context.Context, n int) (string, error) {
	if f.closed {
		return "", fmt.Errorf("attempt to use a closed file")
	}
	if n < 0 {
		return "", fmt.Errorf("not enough memory")
	}
	if err := f.beforeRead(); err != nil {
		return "", err
	}
	if n == 0 {
		// EOF test: peek 1 byte to check if data remains. Any other failure
		// is a read error (ferror in C) and is reported as one.
		if _, err := f.reader.Peek(1); err != nil {
			return "", err
		}
		return "", nil
	}
	return readAtMost(f.reader, n, availableFor(n, f.available))
}

// smallReadLimit is the largest count worth honouring blindly: a buffer this
// size is not an allocation any host notices, so a small read need not pay the
// two syscalls it costs to ask how long the file actually is (a tight
// f:read(1) loop would).
const smallReadLimit = 64 << 10

// availableFor returns the buffer hint for a read of n bytes, consulting the
// (syscall-backed) estimate only when n is large enough to be worth bounding.
func availableFor(n int, estimate func() int) int {
	if n <= smallReadLimit {
		return n
	}
	return estimate()
}

// available estimates how many bytes the file still holds, so ReadBytes can
// size its buffer from what is really there instead of from a count a script
// invents for free. Zero means "unknown" (pipe, device, terminal).
func (f *fullFile) available() int {
	info, err := f.file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return 0
	}
	pos, err := f.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0
	}
	// The descriptor sits ahead of the logical position by whatever the reader
	// has already buffered.
	return clampToInt(info.Size() - pos + int64(f.reader.Buffered()))
}

func clampToInt(n int64) int {
	if n <= 0 {
		return 0
	}
	if n > int64(maxIntValue) {
		return maxIntValue
	}
	return int(n)
}

const maxIntValue = int(^uint(0) >> 1)

// readAtMost reads up to n bytes, growing the destination with the data that
// actually arrives. Sizing the buffer by the requested count instead would let
// f:read(1<<30) on a one-byte file allocate a gigabyte — a Go runtime OOM that
// no pcall can catch, and a process kill in a memory-capped container.
//
// avail is the caller's estimate of how much data is left (0 when unknown): a
// legitimate large read of a large file is then a single allocation of exactly
// the right size, and only an unknown-length stream pays for growing.
func readAtMost(r *bufio.Reader, n, avail int) (string, error) {
	const unknownStart = 64 << 10
	size := n
	if avail > 0 && avail < size {
		size = avail
	} else if avail <= 0 && size > unknownStart {
		size = unknownStart
	}
	if size < 1 {
		size = 1
	}
	buf := make([]byte, 0, size)
	for len(buf) < n {
		if len(buf) == cap(buf) {
			grow := cap(buf) * 2
			if grow > n {
				grow = n
			}
			bigger := make([]byte, len(buf), grow)
			copy(bigger, buf)
			buf = bigger
		}
		nRead, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+nRead]
		if err != nil {
			// Short reads are success: only a request that yielded nothing at
			// all reports the error (C's fread/feof behavior).
			if len(buf) == 0 {
				return "", err
			}
			break
		}
		if nRead == 0 {
			// A reader that yields neither bytes nor an error would spin here;
			// treat it as end of input rather than hang the VM.
			break
		}
	}
	if len(buf) == 0 {
		return "", io.EOF
	}
	return string(buf), nil
}

func (f *fullFile) Write(ctx context.Context, data string) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}

	// Mirror C stdio: writes to a non-writable stream surface the OS error
	// (EBADF on Linux) immediately rather than getting buffered and only
	// failing on flush.
	if !f.writable {
		_, err := f.file.Write([]byte(data))
		if err != nil {
			return err
		}
		return nil
	}

	f.beforeWrite()

	if f.bufMode == "no" {
		// Unbuffered: write directly
		_, err := f.file.Write([]byte(data))
		return err
	}

	// Buffered: write through buffer
	_, err := f.writer.WriteString(data)
	if err != nil {
		return err
	}

	if f.bufMode == "line" {
		// Line buffered: flush if data contains newline
		if strings.ContainsRune(data, '\n') {
			return f.writer.Flush()
		}
	}
	return nil
}

func (f *fullFile) Seek(ctx context.Context, whence string, offset int64) (int64, error) {
	if f.closed {
		return 0, fmt.Errorf("attempt to use a closed file")
	}

	// Flush any buffered writes before seeking
	if f.writer != nil {
		f.writer.Flush()
	}

	var w int
	switch whence {
	case "set":
		w = io.SeekStart
	case "cur":
		w = io.SeekCurrent
	case "end":
		w = io.SeekEnd
	default:
		return 0, fmt.Errorf("invalid option '%s'", whence)
	}

	// When seeking relative to "cur", the OS file descriptor may be ahead
	// of the logical position due to bufio.Reader read-ahead buffering.
	// Adjust the offset to compensate for unconsumed buffered bytes.
	if w == io.SeekCurrent && f.reader != nil {
		buffered := f.reader.Buffered()
		if buffered > 0 {
			offset -= int64(buffered)
		}
	}

	pos, err := f.file.Seek(offset, w)
	if err != nil {
		return 0, err
	}

	// Reset reader after seek so it doesn't return stale buffered data
	f.reader.Reset(f.file)

	return pos, nil
}

func (f *fullFile) Flush(ctx context.Context) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}
	if f.writer != nil {
		err := f.writer.Flush()
		if err != nil {
			// bufio.Writer makes a flush error sticky: it retains the
			// unwritten bytes and returns the same error from every
			// subsequent Write. C stdio behaves differently — fflush()
			// surfaces the device error once, then the stream's buffer is
			// available again, so a later small fwrite() into buffer space
			// succeeds (see the /dev/full flush test in the Lua 5.5 suite).
			// Reset the writer to drop the undeliverable bytes and clear the
			// sticky error, matching that behavior.
			f.writer.Reset(f.file)
		}
		return err
	}
	return nil
}

func (f *fullFile) SetVBuf(ctx context.Context, mode string, size int) error {
	if f.closed {
		return fmt.Errorf("attempt to use a closed file")
	}

	switch mode {
	case "no":
		// Flush any existing buffer
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "no"
		f.writer = nil
	case "full":
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "full"
		f.bufSize = size
		f.writer = newSizedWriter(f.file, size)
	case "line":
		if f.writer != nil {
			f.writer.Flush()
		}
		f.bufMode = "line"
		f.writer = newSizedWriter(f.file, size)
	default:
		return fmt.Errorf("invalid option '%s'", mode)
	}
	return nil
}

// maxVBufSize bounds the buffer a setvbuf request may actually allocate. The
// size is a hint (C stdio is free to ignore it — glibc does), so honouring an
// outsized request literally buys nothing and hands a script an unrecoverable
// allocation.
const maxVBufSize = 1 << 20

// newSizedWriter returns a buffered writer for a setvbuf size request,
// defaulting on non-positive sizes and clamping oversized ones.
func newSizedWriter(w io.Writer, size int) *bufio.Writer {
	if size <= 0 {
		return bufio.NewWriter(w)
	}
	if size > maxVBufSize {
		size = maxVBufSize
	}
	return bufio.NewWriterSize(w, size)
}

func (f *fullFile) Close(ctx context.Context) error {
	if f.closed {
		return fmt.Errorf("cannot close a closed file")
	}
	f.closed = true
	// Flush buffer before closing
	if f.writer != nil {
		f.writer.Flush()
	}
	return f.file.Close()
}

func (f *fullFile) IsClosed(ctx context.Context) bool {
	return f.closed
}

func (f *fullFile) IsStd(ctx context.Context) bool {
	return false
}

// readLine reads a line from the buffered reader.
func (f *fullFile) readLine(keepNewline bool) (string, error) {
	var line strings.Builder
	for {
		b, err := f.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				if line.Len() > 0 {
					return line.String(), nil
				}
				return "", io.EOF
			}
			return "", err
		}
		if b == '\n' {
			if keepNewline {
				line.WriteByte('\n')
			}
			return line.String(), nil
		}
		line.WriteByte(b)
	}
}

// readNumber reads and parses a number from the stream, skipping leading whitespace.
// Consumed whitespace is not restored on failure (matches Lua 5.4 behavior).
func (f *fullFile) readNumber() (string, error) {
	if f.writer != nil {
		f.writer.Flush()
	}

	return readNumberFromBuf(f.reader)
}

// readNumberFromBuf reads a number from a bufio.Reader.
// It skips leading whitespace (which is not restored on failure),
// reads number-like characters, and finds the longest valid prefix
// that parses as a number. Uses Peek to avoid consuming bytes that
// aren't part of the number.
func readNumberFromBuf(reader *bufio.Reader) (string, error) {
	// Skip leading whitespace
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' && b != '\v' && b != '\f' {
			reader.UnreadByte()
			break
		}
	}

	// State-machine scanner matching Lua 5.4's read_number (liolib.c).
	// On failure, all scanned characters are consumed (not restored).
	const maxNumLen = 200 // Lua 5.4's L_MAXLENNUM

	offset := 0
	tooLong := false

	peekByte := func() (byte, bool) {
		if tooLong {
			return 0, false
		}
		peeked, _ := reader.Peek(offset + 1)
		if len(peeked) <= offset {
			return 0, false
		}
		return peeked[offset], true
	}

	// Like the reference's nextc, the length limit only trips when a
	// 201st character would be stored, not when one is merely looked at.
	accept := func() {
		if offset >= maxNumLen {
			tooLong = true
			return
		}
		offset++
	}

	isDigit := func(b byte) bool { return b >= '0' && b <= '9' }
	isHexDigit := func(b byte) bool {
		return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
	}

	fail := func() (string, error) {
		if offset > 0 {
			reader.Discard(offset)
		}
		return "", fmt.Errorf("not a number")
	}

	// Optional sign
	if b, ok := peekByte(); ok && (b == '+' || b == '-') {
		accept()
	}

	// Determine if hex (0x/0X prefix)
	isHex := false
	hasDigits := false
	if b, ok := peekByte(); ok && b == '0' {
		accept()
		hasDigits = true // the '0' itself is a valid digit
		if b2, ok2 := peekByte(); ok2 && (b2 == 'x' || b2 == 'X') {
			accept()
			isHex = true
			hasDigits = false // need at least one hex digit after 0x
		}
	}

	digitCheck := isDigit
	if isHex {
		digitCheck = isHexDigit
	}

	// Read integer part digits
	for {
		b, ok := peekByte()
		if !ok || !digitCheck(b) {
			break
		}
		accept()
		hasDigits = true
	}

	// Optional fractional part
	if b, ok := peekByte(); ok && b == '.' {
		accept()
		for {
			b, ok := peekByte()
			if !ok || !digitCheck(b) {
				break
			}
			accept()
			hasDigits = true
		}
	}

	// Must have at least some digits
	if !hasDigits {
		return fail()
	}

	// Optional exponent (e/E for decimal, p/P for hex)
	expChar := byte('e')
	expCharU := byte('E')
	if isHex {
		expChar = 'p'
		expCharU = 'P'
	}
	if b, ok := peekByte(); ok && (b == expChar || b == expCharU) {
		accept()
		// Optional exponent sign
		if b2, ok2 := peekByte(); ok2 && (b2 == '+' || b2 == '-') {
			accept()
		}
		// Exponent digits (always decimal, even for hex floats)
		hasExpDigits := false
		for {
			b, ok := peekByte()
			if !ok || !isDigit(b) {
				break
			}
			accept()
			hasExpDigits = true
		}
		if !hasExpDigits {
			// Incomplete exponent is a hard failure in Lua 5.4
			return fail()
		}
	}

	// If the number exceeded the maximum length, it's invalid.
	if tooLong {
		return fail()
	}

	// Extract the scanned string and consume it
	peeked, _ := reader.Peek(offset)
	result := string(peeked[:offset])
	reader.Discard(offset)
	return result, nil
}

// stdFile wraps an os.File for standard input/output/error.
// It cannot be closed by the user.
//
// One provider instance is routinely shared by several VMs, each of which may
// be running on its own goroutine, so every stdFile stream is shared too. mu
// serializes the operations that touch the lazily built *bufio.Reader — without
// it two readers race on the reader field and then on the buffer behind it,
// which shows up as corrupt input or an out-of-range slice. It also makes a
// single write one unit, so two VMs writing a line each cannot interleave
// halfway through. This is the same stream-level locking reference Lua does
// with flockfile/funlockfile around its buffered read and write loops.
type stdFile struct {
	mu       sync.Mutex
	file     *os.File
	name     string
	readable bool
	writable bool
	reader   *bufio.Reader
}

// ensureReader returns the stream's buffered reader, building it on first use.
// The caller must hold f.mu.
func (f *stdFile) ensureReader() *bufio.Reader {
	if f.reader == nil {
		f.reader = bufio.NewReader(f.file)
	}
	return f.reader
}

func (f *stdFile) Read(ctx context.Context, format string) (string, error) {
	if !f.readable {
		return "", fmt.Errorf("%s is not readable", f.name)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.ensureReader()
	// Normalize format: strip leading * and check first character
	cleanFmt := strings.TrimPrefix(format, "*")
	if len(cleanFmt) > 0 {
		switch cleanFmt[0] {
		case 'a':
			data, err := io.ReadAll(r)
			if err != nil {
				return "", err
			}
			return string(data), nil
		case 'l':
			return f.readLine(r, false)
		case 'L':
			return f.readLine(r, true)
		case 'n':
			return readNumberFromBuf(r)
		}
	}
	return "", fmt.Errorf("invalid read format: %s", format)
}

func (f *stdFile) ReadBytes(ctx context.Context, n int) (string, error) {
	if !f.readable {
		return "", fmt.Errorf("%s is not readable", f.name)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.ensureReader()
	if n < 0 {
		return "", fmt.Errorf("not enough memory")
	}
	if n == 0 {
		// EOF test: peek 1 byte to check if data remains. Any other failure
		// is a read error (ferror in C) and is reported as one.
		if _, err := r.Peek(1); err != nil {
			return "", err
		}
		return "", nil
	}
	// A standard stream is usually a terminal or a pipe, whose length is not
	// knowable; when it is a redirected regular file the size still bounds the
	// buffer. Either way the count alone never sizes an allocation.
	return readAtMost(r, n, availableFor(n, func() int {
		info, err := f.file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return 0
		}
		pos, err := f.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0
		}
		return clampToInt(info.Size() - pos + int64(r.Buffered()))
	}))
}

func (f *stdFile) Write(ctx context.Context, data string) error {
	if !f.writable {
		return fmt.Errorf("%s is not writable", f.name)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.file.Write([]byte(data))
	return err
}

func (f *stdFile) Seek(ctx context.Context, whence string, offset int64) (int64, error) {
	return 0, fmt.Errorf("cannot seek on %s", f.name)
}

func (f *stdFile) Flush(ctx context.Context) error {
	if !f.writable {
		return fmt.Errorf("%s is not writable", f.name)
	}
	return nil
}

func (f *stdFile) SetVBuf(ctx context.Context, mode string, size int) error {
	return nil // No-op for std files
}

func (f *stdFile) Close(ctx context.Context) error {
	return fmt.Errorf("cannot close standard file")
}

func (f *stdFile) IsClosed(ctx context.Context) bool {
	return false
}

func (f *stdFile) IsStd(ctx context.Context) bool {
	return true
}

func (f *stdFile) readLine(r *bufio.Reader, keepNewline bool) (string, error) {
	var line strings.Builder
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				if line.Len() > 0 {
					return line.String(), nil
				}
				return "", io.EOF
			}
			return "", err
		}
		if b == '\n' {
			if keepNewline {
				line.WriteByte('\n')
			}
			return line.String(), nil
		}
		line.WriteByte(b)
	}
}
