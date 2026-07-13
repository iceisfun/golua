package golua_test

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
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

// thread values are backed by tables internally but must be rejected by
// table-library entry points and ipairs like reference Lua's LUA_TTHREAD.
func TestThreadRejectedByTableLib(t *testing.T) {
	got := runLuaCapture(t, `
local co = coroutine.create(function() end)
print(pcall(table.unpack, co))
print(pcall(table.sort, co))
print(pcall(function() for _ in ipairs(co) do end end))
print(pcall(table.insert, co, 1))
print(pcall(table.move, {1,2,3}, 1, 3, 1, co))`)
	for i, ln := range strings.Split(got, "\n") {
		if !strings.HasPrefix(ln, "false\t") || !strings.Contains(ln, "thread") {
			t.Fatalf("line %d should be a thread-type error: %q (all: %q)", i+1, ln, got)
		}
	}
}

// The compiler never emits the immediate-comparison opcodes (LTI/LEI/GTI/GEI),
// but a binary chunk loaded from the wire can contain them. Their slow paths
// dispatch __lt/__le, and a metamethod that deepens the call stack far enough
// to reallocate it left the cached frame pointer stale, so the conditional
// skip (pc++) was written to dead memory and the comparison took the wrong
// branch. Build the scenario by compiling a normal chunk and patching its
// OP_LT into each immediate form, the way a crafted chunk would arrive.
func TestImmediateCompareMetamethodStaleFrame(t *testing.T) {
	// deep() must recurse through plain CALLs (a tail call reuses its frame
	// and never grows the call stack, which this bug needs).
	const srcTemplate = `
local function deep(n)
	if n ~= 0 then
		local r = deep(n - 1)
		return r
	end
	return 0
end
local mt = {}
mt.__lt = function(a, b) deep(200) return %s end
mt.__le = function(a, b) deep(200) return %s end
local probe = function(t) return t < 5 end
return probe(setmetatable({}, mt))
`
	compile := func(mmResult bool) *compiler.Proto {
		t.Helper()
		lit := "false"
		if mmResult {
			lit = "true"
		}
		src := strings.ReplaceAll(srcTemplate, "%s", lit)
		block, err := parser.Parse("test", src)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		proto, err := compiler.Compile("test", block)
		if err != nil {
			t.Fatalf("compile error: %v", err)
		}
		return proto
	}

	// findLT locates the single OP_LT in the proto tree (probe's `t < 5`).
	var findLT func(p *compiler.Proto) (*compiler.Proto, int)
	findLT = func(p *compiler.Proto) (*compiler.Proto, int) {
		for i, inst := range p.Code {
			if inst.OpCode() == compiler.OP_LT {
				return p, i
			}
		}
		for _, sub := range p.Protos {
			if fp, fi := findLT(sub); fp != nil {
				return fp, fi
			}
		}
		return nil, 0
	}

	for _, op := range []compiler.OpCode{compiler.OP_LTI, compiler.OP_LEI, compiler.OP_GTI, compiler.OP_GEI} {
		// Read the compiled k flag first: the stale-frame write only happens
		// on the pc++ path, which requires the comparison result to differ
		// from k, so pick the metamethod's return value accordingly.
		probeProto, ltIdx := findLT(compile(false))
		if probeProto == nil {
			t.Fatal("no OP_LT found in compiled template")
		}
		lt := probeProto.Code[ltIdx]
		mmResult := lt.K() == 0

		proto := compile(mmResult)
		probeProto, ltIdx = findLT(proto)
		// Same register in A, the constant 5 as the signed-B immediate, same k.
		probeProto.Code[ltIdx] = compiler.ABC(op, lt.A(), 5+compiler.OffsetSC, 0, lt.K())

		v := vm.New()
		stdlib.Open(v)
		results, err := v.Run(proto)
		if err != nil {
			t.Fatalf("%v: runtime error: %v", op, err)
		}
		if len(results) != 1 || !results[0].IsBool() || results[0].AsBool() != mmResult {
			t.Fatalf("%v: comparison took the wrong branch after call-stack growth: got %v want %v",
				op, results, mmResult)
		}
	}
}
