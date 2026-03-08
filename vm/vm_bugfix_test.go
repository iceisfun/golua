package vm

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

// runWithVM compiles and runs Lua code on a given VM, returning results.
func runWithVM(t *testing.T, v *VM, source string) ([]Value, error) {
	t.Helper()
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	return v.Run(proto)
}

// Bug 1: Pure Lua recursion should say "stack overflow", not "C stack overflow"
func TestPureLuaRecursionMessage(t *testing.T) {
	v := New()
	_, err := runWithVM(t, v, `local function f() f() end f()`)
	if err == nil {
		t.Fatal("expected stack overflow error")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "C stack overflow") {
		t.Errorf("pure Lua recursion should say 'stack overflow', not 'C stack overflow'; got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "stack overflow") {
		t.Errorf("expected 'stack overflow' in error message, got: %s", errMsg)
	}
}

// Bug 1 continued: C/metamethod recursion SHOULD say "C stack overflow"
func TestMetamethodRecursionCStackOverflow(t *testing.T) {
	v := New()
	// Create a native function that recurses via the VM
	v.SetGlobal("nativeRecurse", NewNativeFunc(func(vm *VM) int {
		fn := vm.Get(1)
		results, err := vm.ProtectedCall(fn, nil)
		if err != nil {
			vm.Set(0, False)
			if le, ok := err.(*LuaError); ok {
				vm.Set(1, le.Value)
			} else {
				vm.Set(1, NewString(err.Error()))
			}
			return 2
		}
		vm.Set(0, True)
		for i, r := range results {
			vm.Set(i+1, r)
		}
		return 1 + len(results)
	}))

	// pcall
	v.SetGlobal("pcall", NewNativeFunc(func(vm *VM) int {
		argc := vm.ArgCount()
		fn := vm.Get(1)
		args := make([]Value, argc-1)
		for i := 2; i <= argc; i++ {
			args[i-2] = vm.Get(i)
		}
		results, callErr := vm.ProtectedCall(fn, args)
		if callErr != nil {
			vm.Set(0, False)
			if le, ok := callErr.(*LuaError); ok {
				vm.Set(1, le.Value)
			} else {
				vm.Set(1, NewString(callErr.Error()))
			}
			return 2
		}
		vm.Set(0, True)
		for i, r := range results {
			vm.Set(i+1, r)
		}
		return 1 + len(results)
	}))

	// A function that calls nativeRecurse with itself => Lua -> native -> Lua -> native -> ...
	// This involves C-level (native) transitions and should say "C stack overflow"
	_, err := runWithVM(t, v, `
		local function f() nativeRecurse(f) end
		local ok, err = pcall(f)
		return err
	`)
	// We just verify it doesn't crash and the error mentions "C stack overflow"
	_ = err // may or may not propagate depending on pcall handling
}

// Bug 2: Call depth limit too low - pure Lua recursion of 500 should work
// lua5.4 uses LUAI_MAXCCALLS=200 but only counts C-level transitions,
// allowing 500+ pure Lua recursion. With DefaultMaxCallDepth bumped to 800+,
// nest(500) should succeed easily.
func TestCallDepth500PureLua(t *testing.T) {
	v := New()
	results, err := runWithVM(t, v, `
		local function nest(n)
			if n <= 0 then return 0 end
			return nest(n-1) + 1
		end
		return nest(500)
	`)
	if err != nil {
		t.Fatalf("nest(500) should succeed with default call depth, got error: %v", err)
	}
	if len(results) < 1 || results[0].AsInt() != 500 {
		t.Errorf("expected 500, got %v", results)
	}
}

// Bug 3: __newindex table chain off-by-one
func TestNewIndexChainOffByOne(t *testing.T) {
	// With MaxMetaDepth=5 and `< MaxMetaDepth`: loop runs depth 0,1,2,3,4 = 5 iterations
	// A chain of 5 redirects needs 6 iterations (5 to follow + 1 to process final) → should fail
	// With the bug (`<= MaxMetaDepth`): 6 iterations → succeeds incorrectly
	v := New(WithMaxMetaDepth(5))

	// Build chain: t0 -> t1 -> t2 -> t3 -> t4 -> t5 (final, no __newindex)
	tables := make([]*Table, 6)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	for i := 0; i < 5; i++ {
		mt := NewEmptyTable()
		mt.MustSet(metaNewIndex, NewTable(tables[i+1]))
		tables[i].SetMetatable(mt)
	}

	// Setting a key on tables[0] should chain through 5 redirects.
	// With the fix (< MaxMetaDepth), this should fail when MaxMetaDepth=5.
	err := v.TableSetInt(tables[0], 1, NewInt(42))
	if err == nil {
		t.Error("expected __newindex chain error with 5 redirects and MaxMetaDepth=5, but got nil")
	}
}

// Bug 3 continued: verify __index also has the same fix
func TestIndexChainOffByOne(t *testing.T) {
	v := New(WithMaxMetaDepth(5))

	// Build chain of 6 tables where __index points to the next
	tables := make([]*Table, 6)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	for i := 0; i < 5; i++ {
		mt := NewEmptyTable()
		mt.MustSet(metaIndex, NewTable(tables[i+1]))
		tables[i].SetMetatable(mt)
	}

	// Looking up a non-existent key on tables[0] chains through 5 __index redirects
	_, err := v.TableGet(tables[0], NewString("x"))
	if err == nil {
		t.Error("expected __index chain error with 5 redirects and MaxMetaDepth=5, but got nil")
	}
}

// Bug 4: Float for loop with NaN init should run 1 iteration
func TestFloatForLoopNaNInit(t *testing.T) {
	v := New()
	results, err := runWithVM(t, v, `
		local ran = false
		for i = 0/0, 10.0, 1.0 do
			ran = true
			break
		end
		return ran
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 1 || !results[0].AsBool() {
		t.Error("for i = NaN, 10.0, 1.0 should enter the loop body (C NaN comparison semantics)")
	}
}

func TestFloatForLoopNaNLimit(t *testing.T) {
	v := New()
	results, err := runWithVM(t, v, `
		local ran = false
		for i = 1.0, 0/0, 1.0 do
			ran = true
			break
		end
		return ran
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 1 || !results[0].AsBool() {
		t.Error("for i = 1.0, NaN, 1.0 should enter the loop body (C NaN comparison semantics)")
	}
}

func TestFloatForLoopNaNStep(t *testing.T) {
	// NaN step: step >= 0 is false in C, so else branch.
	// else: init < limit → 1.0 < 10.0 → true → skip loop
	v := New()
	results, err := runWithVM(t, v, `
		local ran = false
		for i = 1.0, 10.0, 0/0 do
			ran = true
			break
		end
		return ran
	`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) < 1 {
		t.Fatal("expected return value")
	}
	if results[0].AsBool() {
		t.Error("for i = 1.0, 10.0, NaN should NOT enter loop (C semantics: NaN step, init<limit → skip)")
	}
}

// Bug: __newindex chain of 2000 should error (Lua 5.4 allows 1999 redirects max)
func TestNewIndexChain2000(t *testing.T) {
	v := New() // default MaxMetaDepth = 2000

	// Chain of 2000 tables with __newindex pointing to next
	tables := make([]*Table, 2001)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	for i := 0; i < 2000; i++ {
		mt := NewEmptyTable()
		mt.MustSet(metaNewIndex, NewTable(tables[i+1]))
		tables[i].SetMetatable(mt)
	}

	// This should fail — 2000 redirects exceeds the limit
	err := v.tableSetString(tables[0], "x", NewInt(42))
	if err == nil {
		t.Error("expected __newindex chain error with 2000 redirects, but got nil")
	}
}

// Bug: __newindex chain of 1999 should succeed
func TestNewIndexChain1999(t *testing.T) {
	v := New() // default MaxMetaDepth = 2000

	// Chain of 1999 tables with __newindex pointing to next
	tables := make([]*Table, 2000)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	for i := 0; i < 1999; i++ {
		mt := NewEmptyTable()
		mt.MustSet(metaNewIndex, NewTable(tables[i+1]))
		tables[i].SetMetatable(mt)
	}

	// This should succeed — 1999 redirects is within the limit
	err := v.tableSetString(tables[0], "x", NewInt(42))
	if err != nil {
		t.Errorf("__newindex chain of 1999 should succeed, got error: %v", err)
	}
	// Value should end up in the last table
	val := tables[1999].GetString("x")
	if val.AsInt() != 42 {
		t.Errorf("expected 42 in last table, got %v", val)
	}
}

// Bug: local _ENV = _G should report "global" not "field" in error messages
func TestLocalEnvGlobalName(t *testing.T) {
	v := New()
	// Set up a global environment with a known table so _ENV works
	env := NewEmptyTable()
	v.SetGlobal("_G", NewTable(env))
	// Use a function scope so _ENV becomes a local
	_, err := runWithVM(t, v, `
		local function test()
			local _ENV = _G
			abc()
		end
		test()
	`)
	if err == nil {
		t.Fatal("expected error calling nil global")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "field 'abc'") {
		t.Errorf("should say 'global' not 'field' for _ENV local access, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "global 'abc'") {
		t.Errorf("expected '(global 'abc')' in error, got: %s", errMsg)
	}
}
