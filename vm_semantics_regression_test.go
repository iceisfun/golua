package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// runLuaCaptureOs is runLuaCapture with a DefaultOsProvider so os.* is available.
func runLuaCaptureOs(t *testing.T, source string) string {
	t.Helper()
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New(vm.WithCaptureOutput(true))
	v.SetOsProvider(vm.NewDefaultOsProvider())
	stdlib.Open(v)
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return strings.Join(v.OutputLines(), "\n")
}

// an __index metafield that is a thread must raise, not return nil.
func TestThreadIndex(t *testing.T) {
	got := runLuaCapture(t, `
local co = coroutine.create(function() end)
local t = setmetatable({}, {__index = co})
print(pcall(function() return t.x end))
print(pcall(function() return t[1] end))`)
	lines := strings.Split(got, "\n")
	for _, ln := range lines {
		if !strings.Contains(ln, "attempt to index a thread value") {
			t.Fatalf("thread __index should error: got %q", got)
		}
	}
}

// a recursive erroring __close during pcall error-unwind must be a
// catchable error, not a host-process stack overflow. The VM must stay usable.
func TestRecursiveCloseNoHostCrash(t *testing.T) {
	got := runLuaCapture(t, `
local mt
mt = {__close = function()
  local y <close> = setmetatable({}, mt)
  error("x")
end}
local ok, e = pcall(function()
  local x <close> = setmetatable({}, mt)
end)
print(ok, tostring(e):match("overflow") ~= nil)
print(1 + 1)`)
	if got != "false\ttrue\n2" {
		t.Fatalf("recursive __close should catch a stack-overflow error and VM stay usable: got %q", got)
	}
}

// coroutine.close of a coroutine whose recursive erroring __close
// overflows must report a stack-overflow error (matching reference), not the
// innermost error object, and survive.
func TestRecursiveCloseCoroutine(t *testing.T) {
	got := runLuaCapture(t, `
local mt
mt = {__close = function()
  local y <close> = setmetatable({}, mt)
  error("x")
end}
local co = coroutine.create(function()
  local x <close> = setmetatable({}, mt)
  coroutine.yield()
end)
coroutine.resume(co)
local ok, e = coroutine.close(co)
print(ok, tostring(e):match("overflow") ~= nil)
print("survived")`)
	if got != "false\ttrue\nsurvived" {
		t.Fatalf("coroutine.close of recursive __close should report overflow and survive: got %q", got)
	}
}

// os.time must error when field normalization overflows int tm_year.
func TestOsTimeYearOverflow(t *testing.T) {
	got := runLuaCaptureOs(t, `
print(pcall(os.time, {year=(1<<31)+1899, month=12, day=31, hour=23, min=59, sec=60}))
print(pcall(os.time, {year=(1<<31)+1899, month=12, day=31, hour=23, min=59, sec=59}))`)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected output: %q", got)
	}
	if !strings.Contains(lines[0], "false") || !strings.Contains(lines[0], "cannot be represented") {
		t.Fatalf("os.time overflow should error: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "true\t67768036191705599") {
		t.Fatalf("os.time boundary (sec=59) should succeed: %q", lines[1])
	}
}

// table.remove with the allowed past-the-end position (pos == #list+1) must
// still perform the metamethod-aware read and nil-out of t[pos], like
// reference tremove's unconditional lua_geti/lua_seti pair.
func TestTableRemovePastEnd(t *testing.T) {
	got := runLuaCapture(t, `
local t = setmetatable({10,20,30}, {__len=function() return 2 end})
print(table.remove(t, 3), rawget(t, 3))
local u = setmetatable({}, {__len=function() return 2 end, __index=function(_,k) return k*2 end})
print(table.remove(u, 3))
local gets, sets = {}, {}
local p = setmetatable({}, {
  __len = function() return 0 end,
  __index = function(_, k) gets[#gets+1] = k; return "G"..k end,
  __newindex = function(tt, k, v) sets[#sets+1] = k.."="..tostring(v); rawset(tt, k, v) end,
})
print(table.remove(p, 1), #gets, #sets)`)
	want := "30\tnil\n6\nG1\t1\t1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
