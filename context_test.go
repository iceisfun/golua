package golua_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// runLuaWithContext compiles and runs Lua source with the given context and limits.
// Returns the results and error from execution.
func runLuaWithContext(t *testing.T, source, name string, ctx context.Context, limits vm.Limits) ([]vm.Value, error) {
	t.Helper()

	block, err := parser.Parse(name, source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	proto, err := compiler.Compile(name, block,
		compiler.WithLimits(limits.CompilerLimits))
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

// TestContext_TimeoutInterruptsOsExecute verifies that a blocking native
// os.execute call is surfaced as an execution interruption when the VM's
// context expires, instead of being swallowed as a normal Lua return value.
func TestContext_TimeoutInterruptsOsExecute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	block, err := parser.Parse("test_os_execute_timeout", `return os.execute("sleep 5")`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test_os_execute_timeout", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New(vm.WithContext(ctx))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	v.SetExecProvider(vm.NewDefaultExecProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected interruption from expired context")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected execution interruption, got: %v", err)
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline in error, got: %v", err)
	}
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
	_, err := runLuaWithContext(t, source, "test_cancel_recursion", ctx, vm.Limits{MaxCallDepth: -1})
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
	if !strings.Contains(err.Error(), "stack overflow") {
		t.Fatalf("expected 'stack overflow', got: %v", err)
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
		assert(string.find(err, "stack overflow"), "expected 'stack overflow' in error: " .. err)
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
		assert(string.find(err, "stack overflow"), "expected 'stack overflow' in: " .. tostring(err))
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
		assert(string.find(err, "stack overflow"),
			"expected stack overflow error: " .. err)

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

// ---------------------------------------------------------------------------
// Compiler limit tests
// ---------------------------------------------------------------------------

func genLocals(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "local v%d = %d\n", i, i)
	}
	b.WriteString("return v1\n")
	return b.String()
}

func TestLimits_MaxVars(t *testing.T) {
	// 200 locals — at the default limit, should compile fine.
	source := genLocals(200)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("expected 200 locals to compile, got: %v", err)
	}
}

func TestLimits_MaxVars_Exceeded(t *testing.T) {
	// 201 locals — exceeds the default limit of 200.
	// The parser now enforces this limit too, so the error comes from parsing.
	source := genLocals(201)
	_, err := parser.Parse("test", source)
	if err == nil {
		t.Fatal("expected parse error for 201 locals")
	}
	if !strings.Contains(err.Error(), "too many local variables") {
		t.Fatalf("expected 'too many local variables', got: %v", err)
	}
}

func TestLimits_MaxVars_Override(t *testing.T) {
	// 201 locals with MaxVars=210 — should compile fine.
	// Parser must also use the elevated limit.
	source := genLocals(201)
	block, err := parser.ParseWithMaxVars("test", source, 210)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block, compiler.WithLimits(compiler.CompilerLimits{
		MaxVars: 210,
	}))
	if err != nil {
		t.Fatalf("expected 201 locals with MaxVars=210 to compile, got: %v", err)
	}
}

func genRegsSource(n int) string {
	// Each local uses one register. We need n simultaneous registers.
	// Use a single function with n locals.
	var b strings.Builder
	b.WriteString("local function f()\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "  local v%d = %d\n", i, i)
	}
	b.WriteString("  return v1\nend\nreturn f()\n")
	return b.String()
}

func TestLimits_MaxRegs_Exceeded(t *testing.T) {
	// Force a small MaxRegs and exceed it.
	source := genLocals(50)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block, compiler.WithLimits(compiler.CompilerLimits{
		MaxVars: 50,
		MaxRegs: 30,
	}))
	if err == nil {
		t.Fatal("expected compile error for too many registers")
	}
	if !strings.Contains(err.Error(), "too many registers") {
		t.Fatalf("expected 'too many registers', got: %v", err)
	}
}

func TestLimits_MaxRegs_HardCap(t *testing.T) {
	// Even if user requests MaxRegs=300, it should be clamped to 249.
	source := genLocals(200)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	// 200 locals fit in 249 regs but not in a hypothetical 300 that gets clamped.
	// Just verify the compile succeeds — the hard cap doesn't raise it above 249.
	_, err = compiler.Compile("test", block, compiler.WithLimits(compiler.CompilerLimits{
		MaxRegs: 300, // should be clamped to 249
	}))
	if err != nil {
		t.Fatalf("expected 200 locals to compile with clamped MaxRegs, got: %v", err)
	}
}

func genUpvalSource(n int) string {
	// Create n distinct upvalues in one inner function.
	// Each outer local becomes an upvalue when captured by the inner closure.
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "local u%d = %d\n", i, i)
	}
	b.WriteString("local function f()\n  return u1")
	for i := 2; i <= n; i++ {
		fmt.Fprintf(&b, " + u%d", i)
	}
	b.WriteString("\nend\nreturn f()\n")
	return b.String()
}

func TestLimits_MaxUpvals(t *testing.T) {
	// 100 upvalues — well under the 255 limit and under the 200 local limit.
	source := genUpvalSource(100)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("expected 100 upvalues to compile, got: %v", err)
	}
}

func TestLimits_MaxUpvals_Exceeded(t *testing.T) {
	// Set a low MaxUpvals and exceed it.
	source := genUpvalSource(20)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block, compiler.WithLimits(compiler.CompilerLimits{
		MaxUpvals: 10,
	}))
	if err == nil {
		t.Fatal("expected compile error for too many upvalues")
	}
	if !strings.Contains(err.Error(), "too many upvalues") {
		t.Fatalf("expected 'too many upvalues', got: %v", err)
	}
}

func TestLimits_MaxUpvals_Override(t *testing.T) {
	// 20 upvalues with MaxUpvals=25 — should work.
	source := genUpvalSource(20)
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, err = compiler.Compile("test", block, compiler.WithLimits(compiler.CompilerLimits{
		MaxUpvals: 25,
	}))
	if err != nil {
		t.Fatalf("expected 20 upvalues with MaxUpvals=25 to compile, got: %v", err)
	}
}

func TestLimits_VMReuseAfterInterrupt(t *testing.T) {
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
	block, err := parser.Parse("interrupt_reuse", script)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("interrupt_reuse", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := vm.New(vm.WithLimits(vm.Limits{
		MaxInstructions: 500,
	}))
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err == nil || !strings.Contains(err.Error(), "instruction limit") {
		t.Errorf("Expected instruction limit error, got: %v", err)
	}

	// 3. THE CRITICAL STEP: Reset and Reuse.
	// If the previous run didn't clean up the stack/base pointers,
	// this next run will likely panic or return garbage.
	v.ResetInstructionCount()
	v.SetLimits(vm.Limits{MaxInstructions: 1000})

	block2, err := parser.Parse("interrupt_reuse2", "return 1 + 1")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto2, err := compiler.Compile("interrupt_reuse2", block2)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	res, err := v.Run(proto2)
	if err != nil {
		t.Fatalf("Second run failed (corrupted state?): %v", err)
	}

	if len(res) == 0 || res[0].AsInt() != 2 {
		t.Errorf("VM state corrupted! Expected 2, got %v", res)
	}
}

func TestLimits_MutualTailCallTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	source := `
		local function flop(n)
			return flip(n + 1)
		end
		function flip(n)
			return flop(n + 1)
		end
		return flip(0)
	`
	_, err := runLuaWithContext(t, source, "test_mutual_tailcall", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from context timeout on mutual tail call")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}

func TestLimits_LoadInheritsLimits(t *testing.T) {
	// Verify that load() inside the VM uses the VM's compiler limits.
	source := `
		local code = {}
		for i = 1, 201 do code[#code+1] = "local v"..i.." = "..i end
		code[#code+1] = "return v1"
		local f, err = load(table.concat(code, "\n"))
		assert(f == nil, "expected compile error for 201 locals")
		assert(string.find(err, "too many local"), "expected 'too many local' in: " .. err)
	`
	_, err := runLuaWithContext(t, source, "test_load_inherits_limits", nil, vm.Limits{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
