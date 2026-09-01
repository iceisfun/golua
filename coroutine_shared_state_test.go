package golua_test

// golua models a coroutine as a distinct *VM. Some interpreter state is
// genuinely per-thread (stack, call stack, upvalues) but the rest lives in
// reference Lua's global_State and is shared by every thread of a state: the
// warn flag, the collector's mode/running flags, and the metatables of the
// non-table types. These pin the shared half, in both directions.

import (
	"context"
	"strings"
	"testing"

	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

// capturePrint records print/warn output so a test can assert on warn(), which
// the default provider sends to stderr.
type capturePrint struct {
	prints []string
	warns  []string
}

func (c *capturePrint) Print(ctx context.Context, msg string) { c.prints = append(c.prints, msg) }
func (c *capturePrint) Warn(ctx context.Context, msg string)  { c.warns = append(c.warns, msg) }

func runSharedStateLua(t *testing.T, src string) (*capturePrint, string) {
	t.Helper()
	cap := &capturePrint{}
	v := vm.New()
	if err := v.SetPrintProvider(cap); err != nil {
		t.Fatalf("SetPrintProvider: %v", err)
	}
	if err := v.SetDebugProvider(vm.NewDefaultDebugProvider()); err != nil {
		t.Fatalf("SetDebugProvider: %v", err)
	}
	stdlib.Open(v)
	if _, err := v.Run(compileLua(t, src, "=sharedstate")); err != nil {
		t.Fatalf("run: %v", err)
	}
	return cap, strings.Join(cap.prints, "\n")
}

// warn("@on") is lua_setwarnf on the shared state, so enabling warnings inside
// a coroutine enables them for the whole VM family — and "@off" from a
// coroutine silences the main thread.
func TestCoroutineSharesWarnFlag(t *testing.T) {
	cap, _ := runSharedStateLua(t, `
		coroutine.wrap(function() warn("@on") end)()
		warn("hello from main")
	`)
	if len(cap.warns) != 1 || cap.warns[0] != "Lua warning: hello from main" {
		t.Fatalf("warn(\"@on\") inside a coroutine did not enable warnings for the "+
			"main thread: warns = %q", cap.warns)
	}

	cap, _ = runSharedStateLua(t, `
		warn("@on")
		coroutine.wrap(function() warn("@off") end)()
		warn("should not appear")
	`)
	if len(cap.warns) != 0 {
		t.Fatalf("warn(\"@off\") inside a coroutine did not silence the main thread: "+
			"warns = %q", cap.warns)
	}
}

// collectgarbage("isrunning")/("restart")/("stop") read and write the
// collector's state, which every thread of a state shares.
func TestCoroutineSharesCollectorState(t *testing.T) {
	_, out := runSharedStateLua(t, `
		print("main", collectgarbage("isrunning"))
		coroutine.wrap(function() print("coro", collectgarbage("isrunning")) end)()
		coroutine.wrap(function() collectgarbage("stop") end)()
		print("after", collectgarbage("isrunning"))
	`)
	want := "main\ttrue\ncoro\ttrue\nafter\tfalse"
	if out != want {
		t.Fatalf("collector state is not shared with coroutines:\n got: %q\nwant: %q", out, want)
	}
}

// Metatables for the non-table types are global_State.mt, so one installed
// after a coroutine was created is still visible inside it, and one installed
// inside a coroutine is visible to its parent.
func TestCoroutineSharesTypeMetatables(t *testing.T) {
	_, out := runSharedStateLua(t, `
		local co = coroutine.create(function()
			coroutine.yield()
			print("in coroutine", (3):double())
			debug.setmetatable("", {__index = {shout = function(s) return "S:" .. s end}})
		end)
		coroutine.resume(co)
		debug.setmetatable(0, {__index = {double = function(n) return n * 2 end}})
		print("in main", (3):double())
		coroutine.resume(co)
		print("back in main", ("hi"):shout())
	`)
	want := "in main\t6\nin coroutine\t6\nback in main\tS:hi"
	if out != want {
		t.Fatalf("type metatables are not shared with coroutines:\n got: %q\nwant: %q", out, want)
	}
}
