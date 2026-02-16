package main

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runLuaWithTime(t *testing.T, source, name string) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	v.SetTimeProvider(vm.NewDefaultTimeProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}

func TestTimeNow(t *testing.T) {
	runLuaWithTime(t, `
		local t = time.now()
		assert(type(t) == "number", "time.now() should return a number, got " .. type(t))
		assert(t > 0, "time.now() should be positive")
	`, "test_time_now")
}

func TestTimeSince(t *testing.T) {
	runLuaWithTime(t, `
		local t1 = time.now()
		-- do some work
		local sum = 0
		for i = 1, 100000 do sum = sum + i end
		local elapsed = time.since(t1)
		assert(type(elapsed) == "number", "time.since() should return a number")
		assert(elapsed >= 0, "time.since() should be non-negative")
	`, "test_time_since")
}

func TestTimeNotAvailableWithoutProvider(t *testing.T) {
	block, err := parser.Parse("test", `
		assert(time == nil, "time should not be available without provider")
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	// No time provider set
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
}
