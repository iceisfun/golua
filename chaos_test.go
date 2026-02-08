package main

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func compileScript(t *testing.T, source, name string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return proto
}

func TestChaos_InterruptDuringError(t *testing.T) {
	// 1. A script that does deep recursion with string concatenation.
	// This creates massive pressure on the stack and the allocator.
	script := `
        local function recurse(n)
            local s = "data" .. n
            return recurse(n + 1)
        end
        recurse(1)
    `

	// 2. Run it with a tiny instruction limit — it should hit the limit mid-execution.
	proto := compileScript(t, script, "chaos_interrupt")

	v := vm.New(vm.WithLimits(vm.Limits{
		MaxInstructions: 500,
	}))
	stdlib.Open(v)

	_, err := v.Run(proto)
	if err == nil || !strings.Contains(err.Error(), "instruction limit") {
		t.Errorf("Expected instruction limit error, got: %v", err)
	}

	// 3. THE CRITICAL STEP: Reset and Reuse.
	// If the previous run didn't clean up the stack/base pointers,
	// this next run will likely panic or return garbage.
	v.ResetInstructionCount()
	v.SetLimits(vm.Limits{MaxInstructions: 1000})

	proto2 := compileScript(t, "return 1 + 1", "chaos_reuse")
	res, err := v.Run(proto2)
	if err != nil {
		t.Fatalf("Second run failed (corrupted state?): %v", err)
	}

	if len(res) == 0 || res[0].AsInt() != 2 {
		t.Errorf("VM state corrupted! Expected 2, got %v", res)
	}
}
