package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// runLuaWithContext compiles and runs Lua source with the given context and limits.
// Returns the results and error from execution.
func runLuaWithContext(t *testing.T, source, name string, ctx context.Context, limits vm.Limits) ([]vm.Value, error) {
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
	if ctx != nil {
		v.SetContext(ctx)
	}
	v.SetLimits(limits)
	stdlib.Open(v)

	return v.Run(proto)
}

func TestContext_CancelInfiniteWhile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := runLuaWithContext(t, `while true do end`, "test_cancel_while", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_CancelNumericFor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := runLuaWithContext(t, `for i=1,math.huge do end`, "test_cancel_for", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_CancelRecursion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local function recurse(n)
			local x = recurse(n + 1) -- not a tail call
			return x
		end
		recurse(0)
	`
	_, err := runLuaWithContext(t, source, "test_cancel_recursion", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_CancelTailCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local function loop()
			return loop()
		end
		loop()
	`
	_, err := runLuaWithContext(t, source, "test_cancel_tailcall", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_Deadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := runLuaWithContext(t, `while true do end`, "test_deadline", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from deadline")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_NilDefault(t *testing.T) {
	source := `
		local sum = 0
		for i = 1, 100 do
			sum = sum + i
		end
		assert(sum == 5050, "expected 5050, got " .. tostring(sum))
	`
	_, err := runLuaWithContext(t, source, "test_nil_default", nil, vm.Limits{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContext_CoroutineInheritance(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local co = coroutine.create(function()
			while true do end
		end)
		local ok, err = coroutine.resume(co)
		if not ok then
			error(err)
		end
	`
	_, err := runLuaWithContext(t, source, "test_coroutine_inherit", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context in coroutine")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestContext_CoroutineResumeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	source := `
		local co = coroutine.create(function()
			while true do
				coroutine.yield()
			end
		end)
		while true do
			local ok, err = coroutine.resume(co)
			if not ok then
				error(err)
			end
		end
	`
	_, err := runLuaWithContext(t, source, "test_coroutine_resume_cancelled", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestLimits_MaxCallDepth(t *testing.T) {
	source := `
		local function recurse(n)
			local x = recurse(n + 1)
			return x
		end
		recurse(0)
	`
	limits := vm.Limits{MaxCallDepth: 10}
	_, err := runLuaWithContext(t, source, "test_max_call_depth", nil, limits)
	if err == nil {
		t.Fatal("expected call stack overflow error")
	}
	if !strings.Contains(err.Error(), "call stack overflow") {
		t.Fatalf("expected 'call stack overflow', got: %v", err)
	}
}

func TestLimits_MaxCallDepth_Pcall(t *testing.T) {
	source := `
		local function recurse(n)
			local x = recurse(n + 1)
			return x
		end
		local ok, err = pcall(recurse, 0)
		assert(not ok, "expected pcall to fail")
		assert(type(err) == "string", "expected error string, got " .. type(err))
		assert(string.find(err, "call stack overflow"), "expected 'call stack overflow' in error: " .. err)
	`
	limits := vm.Limits{MaxCallDepth: 10}
	_, err := runLuaWithContext(t, source, "test_max_call_depth_pcall", nil, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLimits_MaxStackSlots(t *testing.T) {
	source := `
		local function recurse(n)
			local x = recurse(n + 1)
			return x
		end
		local ok, err = pcall(recurse, 0)
		assert(not ok, "expected pcall to fail")
		assert(type(err) == "string", "expected error string, got " .. type(err))
		assert(string.find(err, "stack overflow"), "expected 'stack overflow' in error: " .. err)
	`
	limits := vm.Limits{MaxStackSlots: 2000}
	_, err := runLuaWithContext(t, source, "test_max_stack_slots", nil, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLimits_MaxInstructions(t *testing.T) {
	source := `
		while true do end
	`
	limits := vm.Limits{MaxInstructions: 100}
	_, err := runLuaWithContext(t, source, "test_max_instructions", nil, limits)
	if err == nil {
		t.Fatal("expected instruction limit error")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("expected 'instruction limit exceeded', got: %v", err)
	}
}

func TestLimits_CoroutineInheritance(t *testing.T) {
	source := `
		local co = coroutine.create(function()
			local function recurse(n)
				local x = recurse(n + 1)
				return x
			end
			recurse(0)
		end)
		local ok, err = coroutine.resume(co)
		assert(not ok, "expected coroutine to fail")
		assert(string.find(err, "call stack overflow"), "expected 'call stack overflow' in: " .. tostring(err))
	`
	limits := vm.Limits{MaxCallDepth: 10}
	_, err := runLuaWithContext(t, source, "test_limits_coroutine_inherit", nil, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContext_FunctionalOption(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	block, err := parser.Parse("test", `while true do end`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New(vm.WithContext(ctx))
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestLimits_FunctionalOption(t *testing.T) {
	block, err := parser.Parse("test", `while true do end`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New(vm.WithLimits(vm.Limits{MaxInstructions: 50}))
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected instruction limit error")
	}
	if !strings.Contains(err.Error(), "instruction limit exceeded") {
		t.Fatalf("expected 'instruction limit exceeded', got: %v", err)
	}
}

func TestContext_VMReuse(t *testing.T) {
	// First run: cancel a loop
	ctx1, cancel1 := context.WithCancel(context.Background())

	block, err := parser.Parse("test", `
		local sum = 0
		for i = 1, math.huge do sum = sum + 1 end
	`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New()
	stdlib.Open(v)
	v.SetContext(ctx1)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel1()
	}()

	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	// Second run: fresh context, should work
	v.SetContext(context.Background())
	v.ResetInstructionCount()

	block2, err := parser.Parse("test2", `return 42`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto2, err := compiler.Compile("test2", block2)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	results, err := v.Run(proto2)
	if err != nil {
		t.Fatalf("unexpected error on reuse: %v", err)
	}
	if len(results) == 0 || results[0].AsInt() != 42 {
		t.Fatalf("expected 42, got: %v", results)
	}
}

func TestContext_ErrorUnwindsCleanly(t *testing.T) {
	// Verify pcall catches call depth errors and VM state remains consistent.
	// Use MaxCallDepth (not MaxInstructions) so the outer code can still
	// execute after pcall catches the inner error.
	source := `
		local function recurse(n)
			local x = recurse(n + 1)
			return x
		end
		local ok, err = pcall(recurse, 0)
		assert(not ok, "expected pcall to catch error")
		assert(type(err) == "string", "expected error string")
		assert(string.find(err, "call stack overflow"),
			"expected call stack overflow error: " .. err)

		-- VM state should be consistent after pcall catches the error
		local sum = 0
		for i = 1, 10 do
			sum = sum + i
		end
		assert(sum == 55, "expected 55 after recovery, got " .. tostring(sum))
		return sum
	`
	limits := vm.Limits{MaxCallDepth: 15}
	results, err := runLuaWithContext(t, source, "test_error_unwinds", nil, limits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || results[0].AsInt() != 55 {
		t.Fatalf("expected 55, got: %v", results)
	}
}
