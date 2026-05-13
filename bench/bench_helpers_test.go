package bench

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

// scriptDir returns the absolute path to the bench/scripts directory.
func scriptDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "scripts")
}

// loadScript reads a Lua script from the scripts directory.
func loadScript(b *testing.B, name string) string {
	b.Helper()
	data, err := os.ReadFile(filepath.Join(scriptDir(), name))
	if err != nil {
		b.Fatal(err)
	}
	return string(data)
}

// compileScript parses and compiles a Lua source string.
func compileScript(b *testing.B, name, source string) *compiler.Proto {
	b.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		b.Fatalf("parse %s: %v", name, err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		b.Fatalf("compile %s: %v", name, err)
	}
	return proto
}

// newVM creates a fresh VM with stdlib loaded.
func newVM() *vm.VM {
	v := vm.New()
	stdlib.Open(v)
	return v
}

// runProto executes a compiled prototype on a fresh VM.
func runProto(b *testing.B, proto *compiler.Proto) {
	b.Helper()
	v := newVM()
	if _, err := v.Run(proto); err != nil {
		b.Fatal(err)
	}
}

// benchmarkScript is the common pattern: compile once, run N times.
func benchmarkScript(b *testing.B, name string) {
	source := loadScript(b, name)
	proto := compileScript(b, name, source)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runProto(b, proto)
	}
}

// registerBenchHelpers injects the bench.* helper table into a VM.
func registerBenchHelpers(v *vm.VM) {
	t := vm.NewEmptyTable()

	// bench.gc() - force garbage collection
	t.Set(vm.NewString("gc"), vm.NewNativeFunc(func(v *vm.VM) int {
		runtime.GC()
		return 0
	}))

	// bench.consume(value) - prevent dead-code elimination
	var sink vm.Value
	t.Set(vm.NewString("consume"), vm.NewNativeFunc(func(v *vm.VM) int {
		sink = v.Get(1)
		_ = sink
		return 0
	}))

	v.SetGlobal("bench", vm.NewTable(t))
}
