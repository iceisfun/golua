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

// TestLuaFiles runs all .lua test files in the tests directory.
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
				runErr = r.(error)
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
	files, err := filepath.Glob("test_*.lua")
	if err != nil {
		b.Fatalf("Failed to glob lua files: %v", err)
	}

	for _, file := range files {
		file := file
		name := strings.TrimSuffix(file, ".lua")

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
