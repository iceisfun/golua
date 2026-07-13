package golua_test

import (
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func TestCoroutineAPIRegression_RejectsForgedThreadTables(t *testing.T) {
	source := `
		local fake = { __coroutine_id = 0 }
		local ok1, err1 = pcall(coroutine.status, fake)
		local ok2, err2 = pcall(coroutine.resume, fake)
		local ok3, err3 = pcall(coroutine.close, fake)
		assert(ok1 == false and tostring(err1):find("thread expected, got table", 1, true), tostring(err1))
		assert(ok2 == false and tostring(err2):find("thread expected, got table", 1, true), tostring(err2))
		assert(ok3 == false and tostring(err3):find("thread expected, got table", 1, true), tostring(err3))
	`
	runLuaSource(t, source, "test_coroutine_reject_forged_thread_table")
}

func TestCoroutineAPIRegression_IsYieldableRemainsTrueAfterClose(t *testing.T) {
	source := `
		local co = coroutine.create(function()
			coroutine.yield("pause")
		end)
		assert(coroutine.resume(co))
		assert(coroutine.close(co))
		assert(coroutine.status(co) == "dead")
		assert(coroutine.isyieldable(co) == true)
	`
	runLuaSource(t, source, "test_coroutine_isyieldable_after_close")
}

// os.exit inside a coroutine must propagate the exit sentinel through the
// resume boundary (and nested coroutines), not surface as a catchable
// resume error that leaves the process running.
func TestOsExitInsideCoroutine(t *testing.T) {
	src := `
local inner = coroutine.create(function() os.exit(7) end)
local outer = coroutine.create(function() coroutine.resume(inner) end)
pcall(function() coroutine.resume(outer) end)
return "unreachable"`
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	v := vm.New()
	if err := v.SetOsProvider(vm.NewDefaultOsProvider()); err != nil {
		t.Fatal(err)
	}
	if err := v.SetExitHandler(vm.NewDefaultExitHandler()); err != nil {
		t.Fatal(err)
	}
	stdlib.Open(v)
	defer func() {
		r := recover()
		exitErr, ok := r.(*vm.LuaExitError)
		if !ok {
			t.Fatalf("expected LuaExitError panic, got %v", r)
		}
		if exitErr.Code != 7 {
			t.Fatalf("exit code = %d, want 7", exitErr.Code)
		}
	}()
	results, err := v.Run(proto)
	t.Fatalf("Run returned instead of exiting: %v %v", results, err)
}
