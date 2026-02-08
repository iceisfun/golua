package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

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
			runLuaTest(t, file)
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
			runLuaTest(t, file)
		})
	}
}

// runLuaTest compiles and runs a single Lua test file.
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

	// Run
	v := vm.New()
	stdlib.Open(v)

	// Capture panics from assert() failures
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr, _ = r.(error)
				if runErr == nil {
					// panic was not an error type
					t.Fatalf("Runtime panic: %v", r)
				}
			}
		}()
		_, runErr = v.Run(proto)
	}()

	if runErr != nil {
		t.Fatalf("Runtime error: %v", runErr)
	}
}

func compileLua(name, source string) (*compiler.Proto, error) {
	block, err := parser.Parse(name, source)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(name, block)
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
