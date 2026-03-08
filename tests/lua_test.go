package tests

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// full enables resource-intensive tests (test_heavy_*) that may use large
// amounts of memory and CPU. Pass -full to include them.
var full = flag.Bool("full", false, "run resource-intensive heavy tests")

// luaTestTimeout is the maximum time a single Lua test file may run before
// being terminated. Prevents deadlocks from stalling the entire test suite.
const luaTestTimeout = 30 * time.Second

// TestLuaFiles runs all .lua test files in the tests directory (root level).
// Files are categorized by prefix:
//   - test_*.lua  : Regular tests that should pass
//   - broken_*.lua: Known broken tests (skipped, tracked as issues)
func TestLuaFiles(t *testing.T) {
	files, err := filepath.Glob("*.lua")
	if err != nil {
		t.Fatalf("Failed to glob lua files: %v", err)
	}

	for _, file := range files {
		file := file // capture for closure
		name := strings.TrimSuffix(file, ".lua")

		if strings.HasPrefix(file, "broken_") {
			t.Run(name, func(t *testing.T) {
				t.Skip("Known broken test - see file for details")
			})
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(file, "test_heavy_") && !*full {
				t.Skip("heavy test requires -full flag")
			}
			if needsFullIo(file) {
				runLuaTestFullIo(t, file)
			} else {
				runLuaTest(t, file)
			}
		})
	}
}

// TestStdlib runs all .lua files in the stdlib/ directory.
// These are regression tests for standard library functionality
// and must always pass.
func TestStdlib(t *testing.T) {
	runLuaDir(t, "stdlib")
}

// TestStress runs all .lua files in the stress/ directory.
// These are workload/performance tests for table, loop, and allocation stability.
func TestStress(t *testing.T) {
	runLuaDir(t, "stress")
}

// TestBroken lists all .lua files in the broken/ directory as skipped tests.
// These represent known missing features or known failures.
func TestBroken(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("broken", "*.lua"))
	if err != nil || len(files) == 0 {
		return
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".lua")

		t.Run(name, func(t *testing.T) {
			t.Skip("Known broken test - see file for details")
		})
	}
}

// runLuaDir discovers and runs all .lua files in a subdirectory.
func runLuaDir(t *testing.T, dir string) {
	files, err := filepath.Glob(filepath.Join(dir, "*.lua"))
	if err != nil {
		t.Fatalf("Failed to glob %s/*.lua: %v", dir, err)
	}
	if len(files) == 0 {
		t.Skipf("No .lua files found in %s/", dir)
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".lua")

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(filepath.Base(file), "test_heavy_") && !*full {
				t.Skip("heavy test requires -full flag")
			}
			if needsFullIo(file) {
				runLuaTestFullIo(t, file)
			} else {
				runLuaTest(t, file)
			}
		})
	}
}

// runLuaTest compiles and runs a single Lua test file with timeout protection.
func runLuaTest(t *testing.T, filename string) {
	t.Helper()

	// Read file
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filename, err)
	}

	// Compile
	proto, err := compileLua(filename, string(source))
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), luaTestTimeout)
	defer cancel()

	// Run
	v := vm.New(vm.WithContext(ctx))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	v.SetIoProvider(vm.NewJailedIoProvider("."))
	v.SetCodeProvider(vm.NewDirCodeProvider(".", vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	v.SetExecProvider(vm.NewDefaultExecProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	stdlib.Open(v)

	// Run in goroutine with panic recovery and timeout
	resultCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if _, isExit := r.(*vm.LuaExitError); isExit {
					resultCh <- nil // os.exit is normal termination in tests
					return
				}
				if e, ok := r.(error); ok {
					resultCh <- e
				} else {
					resultCh <- fmt.Errorf("runtime panic: %v", r)
				}
			}
		}()
		_, runErr := v.Run(proto)
		resultCh <- runErr
	}()

	select {
	case runErr := <-resultCh:
		if runErr != nil {
			t.Fatalf("Runtime error: %v", runErr)
		}
	case <-time.After(luaTestTimeout + 2*time.Second):
		cancel()
		t.Fatalf("Test timed out after %v (possible deadlock)", luaTestTimeout)
	}
}

// runLuaTestFullIo is like runLuaTest but uses FullIoProvider for tests that
// need write access, standard file handles, os.tmpname, and os.remove.
func runLuaTestFullIo(t *testing.T, filename string) {
	t.Helper()

	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filename, err)
	}

	proto, err := compileLua(filename, string(source))
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), luaTestTimeout)
	defer cancel()

	v := vm.New(vm.WithContext(ctx))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	v.SetIoProvider(vm.NewFullIoProvider("."))
	v.SetCodeProvider(vm.NewDirCodeProvider(".", vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	v.SetExecProvider(vm.NewDefaultExecProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	stdlib.Open(v)

	// Run in goroutine with panic recovery and timeout
	resultCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if _, isExit := r.(*vm.LuaExitError); isExit {
					resultCh <- nil // os.exit is normal termination in tests
					return
				}
				if e, ok := r.(error); ok {
					resultCh <- e
				} else {
					resultCh <- fmt.Errorf("runtime panic: %v", r)
				}
			}
		}()
		_, runErr := v.Run(proto)
		resultCh <- runErr
	}()

	select {
	case runErr := <-resultCh:
		if runErr != nil {
			t.Fatalf("Runtime error: %v", runErr)
		}
	case <-time.After(luaTestTimeout + 2*time.Second):
		cancel()
		t.Fatalf("Test timed out after %v (possible deadlock)", luaTestTimeout)
	}
}

// needsFullIo returns true for test files that require FullIoProvider
// (write access, standard file handles, os.tmpname, os.remove).
func needsFullIo(filename string) bool {
	base := filepath.Base(filename)
	if strings.HasPrefix(base, "io_") {
		return true
	}
	// Tests that reference io.stdin/stdout/stderr as values
	switch base {
	case "test_next_all_key_types.lua",
		"test_error_named_objects.lua",
		"test_dofile_loadfile.lua",
		"test_bom_load.lua",
		"test_read0_eof.lua",
		"test_read_n_hex_float.lua",
		"test_file_metatable.lua":
		return true
	}
	return false
}

func compileLua(name, source string) (*compiler.Proto, error) {
	// Use "@" prefix for source names to match Lua 5.4 convention
	// where file-based sources are stored as "@filename".
	srcName := name
	if len(srcName) > 0 && srcName[0] != '@' && srcName[0] != '=' {
		srcName = "@" + srcName
	}
	block, err := parser.Parse(name, source)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(srcName, block)
}

// TestDoctest runs doctest-style Lua files from tests/doctest/.
// These files use --> comments to specify expected print() output:
//
//	--> =exact output    (exact match after tab-joining print args)
//	--> ~regex pattern   (regex match against the output line)
//
// The harness captures all print() output via WithCaptureOutput and validates
// it against directives in order. Lines without --> are not checked.
func TestDoctest(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("doctest", "*.lua"))
	if err != nil || len(files) == 0 {
		t.Skip("No doctest files found in doctest/")
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".lua")

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runLuaDoctest(t, file)
		})
	}
}

// TestProposed runs proposed test files from ../proposed_tests/ using the
// doctest harness. This is a staging area for new tests before they are
// promoted to tests/doctest/. Skipped if no files exist.
func TestProposed(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "proposed_tests", "*.lua"))
	if err != nil || len(files) == 0 {
		t.Skip("No proposed test files found")
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".lua")

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runLuaDoctest(t, file)
		})
	}
}

// directive represents one --> expected-output comment.
type directive struct {
	line    int    // source line number (1-based)
	exact   bool   // true for =exact, false for ~regex
	pattern string // the expected text or regex
}

// parseDirectives extracts --> directives from Lua source.
func parseDirectives(source string) []directive {
	var dirs []directive
	for i, line := range strings.Split(source, "\n") {
		// Look for --> anywhere in the line (typically in a comment)
		idx := strings.Index(line, "-->")
		if idx == -1 {
			continue
		}
		rest := strings.TrimSpace(line[idx+3:])
		if len(rest) == 0 {
			continue
		}
		switch rest[0] {
		case '=':
			dirs = append(dirs, directive{line: i + 1, exact: true, pattern: rest[1:]})
		case '~':
			dirs = append(dirs, directive{line: i + 1, exact: false, pattern: rest[1:]})
		default:
			dirs = append(dirs, directive{line: i + 1, exact: true, pattern: rest})
		}
	}
	return dirs
}

// runLuaDoctest compiles and runs a Lua file in a goroutine with timeout,
// VM limits, and doctest helper functions. Captures print() output via
// WithCaptureOutput and validates it against --> directives in the source.
func runLuaDoctest(t *testing.T, filename string) {
	t.Helper()

	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", filename, err)
	}

	dirs := parseDirectives(string(source))
	if len(dirs) == 0 {
		t.Fatalf("No --> directives found in %s", filename)
	}

	proto, err := compileLua(filename, string(source))
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Set up context with timeout
	timeout := defaultDoctestTimeout
	ctx, cancel := context.WithCancel(context.Background())
	deadline := time.Now().Add(timeout)
	timer := time.AfterFunc(timeout, cancel)

	cfg := &doctestConfig{
		defaultTimeout: timeout,
		deadline:       deadline,
		cancel:         cancel,
		timer:          timer,
	}
	defer timer.Stop()
	defer cancel()

	// Create VM with output capture, context, and limits
	v := vm.New(
		vm.WithCaptureOutput(true),
		vm.WithContext(ctx),
		vm.WithLimits(doctestLimits()),
	)
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	stdlib.Open(v)
	registerDoctestHelpers(v, cfg)

	// Run in goroutine with panic recovery
	resultCh := make(chan doctestResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- classifyPanic(r)
			}
		}()
		_, runErr := v.Run(proto)
		if runErr != nil {
			resultCh <- classifyPanic(runErr)
		} else {
			resultCh <- doctestResult{kind: resultSuccess}
		}
	}()

	// Wait for result or safety timeout (default + 2s for true deadlocks)
	var result doctestResult
	select {
	case result = <-resultCh:
		// normal completion
	case <-time.After(timeout + 2*time.Second):
		result = doctestResult{
			kind: resultTimeout,
			err:  fmt.Errorf("safety timeout: goroutine did not respond after %v", timeout+2*time.Second),
		}
	}

	// Validate captured output against directives
	outputLines := v.OutputLines()

	// Report errors by kind
	switch result.kind {
	case resultLuaError:
		t.Errorf("Runtime error (after %d output lines): %v", len(outputLines), result.err)
	case resultVMPanic:
		t.Errorf("VM PANIC (bug): %v", result.err)
	case resultTimeout:
		t.Errorf("TIMEOUT: %v", result.err)
	}

	if len(outputLines) < len(dirs) {
		t.Errorf("Expected at least %d output lines, got %d", len(dirs), len(outputLines))
		for i, line := range outputLines {
			t.Logf("  output[%d]: %q", i, line)
		}
	}

	for i, dir := range dirs {
		if i >= len(outputLines) {
			t.Errorf("Line %d: missing output for directive %q", dir.line, dir.pattern)
			continue
		}
		got := outputLines[i]
		if dir.exact {
			if got != dir.pattern {
				t.Errorf("Line %d: output mismatch\n  want: %q\n  got:  %q", dir.line, dir.pattern, got)
			}
		} else {
			re, err := regexp.Compile(dir.pattern)
			if err != nil {
				t.Errorf("Line %d: invalid regex %q: %v", dir.line, dir.pattern, err)
				continue
			}
			if !re.MatchString(got) {
				t.Errorf("Line %d: output does not match pattern\n  pattern: %s\n  got:     %q", dir.line, dir.pattern, got)
			}
		}
	}

	if len(outputLines) > len(dirs) {
		t.Errorf("Got %d extra output lines beyond %d directives", len(outputLines)-len(dirs), len(dirs))
		for i := len(dirs); i < len(outputLines); i++ {
			t.Logf("  extra[%d]: %q", i, outputLines[i])
		}
	}
}

// BenchmarkLuaFiles allows benchmarking individual test files.
func BenchmarkLuaFiles(b *testing.B) {
	// Collect files from root and stdlib/
	var files []string
	for _, pattern := range []string{"test_*.lua", "stdlib/*.lua", "stress/*.lua"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		files = append(files, matches...)
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(file, ".lua")
		name = strings.ReplaceAll(name, string(filepath.Separator), "/")

		b.Run(name, func(b *testing.B) {
			source, err := os.ReadFile(file)
			if err != nil {
				b.Fatalf("Failed to read %s: %v", file, err)
			}

			proto, err := compileLua(file, string(source))
			if err != nil {
				b.Fatalf("Compilation failed: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v := vm.New()
				stdlib.Open(v)
				v.Run(proto)
			}
		})
	}
}
