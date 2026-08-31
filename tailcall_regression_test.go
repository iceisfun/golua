package golua_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/vm"
)

// A tail call reuses the caller's frame, so the frame's register window changes
// size under it. The callee's window has to be established before it runs —
// otherwise everything that builds a frame at the current stack top (a
// metamethod, a coercion, a nested call) lands in the middle of the callee's
// own live registers.
func TestTailCallEstablishesCalleeRegisterWindow(t *testing.T) {
	got := runLuaCapture(t, `
local probe = setmetatable({}, {__index = function() return "MM" end})

local function big()
  local a, b, c, d, e = 1, 2, 3, 4, 5
  local x = probe.missing
  return x, a, b, c, d, e
end

local function small() return big() end

print(small())`)
	if want := "MM\t1\t2\t3\t4\t5"; got != want {
		t.Errorf("tail-callee locals after a metamethod: got %q, want %q", got, want)
	}
}

// The same defect in the shape it actually turns up in: a dispatcher that ends
// in "return handlers[k](...)" is a narrow frame tail-calling a wider one.
func TestTailCallEstablishesCalleeRegisterWindowForADispatcher(t *testing.T) {
	got := runLuaCapture(t, `
local probe = setmetatable({}, {__index = function() return "MM" end})

local handlers = {}
function handlers.run(...)
  local p, q, r, s, t = 10, 20, 30, 40, 50
  local z = probe.missing
  return z, p, q, r, s, t
end

local function dispatch(k, ...) return handlers[k](...) end

print(dispatch("run"))`)
	if want := "MM\t10\t20\t30\t40\t50"; got != want {
		t.Errorf("tail-callee locals in a dispatcher: got %q, want %q", got, want)
	}
}

// A tail-callee whose register window is wider than the frame it takes over
// needs that stack space allocated for it. Recursing first walks the frame base
// upward so the wide window crosses the end of the allocated stack at some
// depth.
func TestTailCallGrowsStackForAWiderCallee(t *testing.T) {
	var b strings.Builder
	b.WriteString("local function big()\n  local ")
	for i := 1; i <= 190; i++ {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteString("l")
		b.WriteString(strconv.Itoa(i))
	}
	b.WriteString(" = ")
	for i := 1; i <= 190; i++ {
		if i > 1 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Itoa(i))
	}
	b.WriteString("\n  return l1 + l190\nend\n")
	b.WriteString(`
local function small() return big() end

local function rec(n)
  if n == 0 then return small() end
  local r = rec(n - 1)
  return r
end

local failures = 0
for i = 0, 60 do
  local ok, v = pcall(rec, i)
  if not ok or v ~= 191 then failures = failures + 1 end
end
print(failures)`)
	if got := runLuaCapture(t, b.String()); got != "0" {
		t.Errorf("tail call into a wider frame: %s depths failed, want 0", got)
	}
}

// A non-vararg function that took over a vararg function's frame by tail call
// has no varargs of its own, and must not show the ones the dead caller had.
func TestTailCallClearsTheCallersVarargs(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function target(a)
			assert(debug.getlocal(1, -1) == nil, "vararg -1 leaked")
			assert(debug.getlocal(1, -2) == nil, "vararg -2 leaked")
			assert(debug.getinfo(1, "u").isvararg == false)
		end
		local function src(...) return target(1) end
		src("X", "Y")
	`
	runLuaWithDebug(t, source, "test_tailcall_varargs", provider)
}

// A tail call into a native function still runs the native as a call of its
// own, so it gets a call and a return event; and the Lua frame it replaced
// retires at that point, so that frame's return event fires too. A hook that
// counts call/return depth must come back to where it started.
func TestTailCallIntoNativeFiresBalancedHooks(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local log = {}
		local function a() return select('#', 1) end
		debug.sethook(function(ev)
			local i = debug.getinfo(2, "nS")
			log[#log + 1] = ev .. " " .. tostring(i and i.what) .. " " .. tostring(i and i.name)
		end, "cr")
		a()
		debug.sethook()
		local got = table.concat(log, "|")
		local want = "return C sethook|call Lua a|call C select|return C select|return Lua a|call C sethook"
		assert(got == want, got)

		local depth = 0
		local function fmt() return string.format("%d", 1) end
		debug.sethook(function(ev)
			if ev == "call" or ev == "tail call" then depth = depth + 1
			elseif ev == "return" then depth = depth - 1 end
		end, "cr")
		for i = 1, 4 do fmt() end
		debug.sethook()
		assert(depth == 0, "unbalanced hook depth: " .. tostring(depth))
	`
	runLuaWithDebug(t, source, "test_tailcall_native_hooks", provider)
}

// A return hook runs arbitrary Lua, and any Lua function that returns a value
// from inside it leaves through OP_RETURN — which writes the very buffer a tail
// call into a native uses to carry its own results out. Whatever the hook does,
// the caller has to see the native's results.
func TestTailCallResultsSurviveAReturnHook(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local function noise() return "AAA", "BBB", "CCC" end

		-- a result list small enough to ride the shared return buffer
		local function four() return select(2, "p", "q", "r", "s") end
		debug.sethook(function() noise() end, "r")
		local a = table.pack(four())
		debug.sethook()
		assert(a.n == 3, "select count: " .. tostring(a.n))
		assert(a[1] == "q" and a[2] == "r" and a[3] == "s",
			"select results: " .. table.concat(a, ",", 1, a.n))

		-- "return pcall(f)" is the shape this turns up in
		local function boom() return pcall(function() error("boom", 0) end) end
		debug.sethook(function() noise() end, "cr")
		local ok, err = boom()
		debug.sethook()
		assert(ok == false, "pcall status: " .. tostring(ok))
		assert(err == "boom", "pcall message: " .. tostring(err))

		-- more results than that buffer holds, and more than the native frame
		-- was grown for
		local big = {}
		for i = 1, 20 do big[i] = i * 10 end
		local function many() return table.unpack(big) end
		debug.sethook(function() noise() end, "r")
		local b = table.pack(many())
		debug.sethook()
		assert(b.n == 20, "unpack count: " .. tostring(b.n))
		for i = 1, 20 do
			assert(b[i] == i * 10, "unpack result " .. i .. ": " .. tostring(b[i]))
		end

		-- no results at all
		local sink = {}
		local function nothing() return table.insert(sink, "v") end
		debug.sethook(function() noise() end, "r")
		local c = table.pack(nothing())
		debug.sethook()
		assert(c.n == 0, "insert result count: " .. tostring(c.n))
		assert(sink[1] == "v", "insert did not run")

		-- the hook body itself tail-calls a native
		local function inner() return select(2, "i", "j", "k") end
		debug.sethook(function() inner() end, "r")
		local d = table.pack((function() return select(2, 1, 2, 3, 4) end)())
		debug.sethook()
		assert(d.n == 3 and d[1] == 2 and d[2] == 3 and d[3] == 4,
			"nested tail call results: " .. table.concat(d, ",", 1, d.n))

		-- every arity across the shared buffer's width
		local src = {}
		for i = 1, 12 do src[i] = i end
		for want = 0, 12 do
			local f = function() return table.unpack(src, 1, want) end
			debug.sethook(function() noise() end, "cr")
			local e = table.pack(f())
			debug.sethook()
			assert(e.n == want, "arity " .. want .. ": got " .. tostring(e.n))
			for i = 1, want do
				assert(e[i] == i, "arity " .. want .. " result " .. i .. ": " .. tostring(e[i]))
			end
		end
	`
	runLuaWithDebug(t, source, "test_tailcall_hook_results", provider)
}

// A tail-called native can install the hook itself, so whether one will run is
// only known once it has returned: its own return event still fires, and the
// results of the tail calls that follow are still its caller's.
func TestTailCallResultsSurviveAHookTheNativeInstalls(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local events = {}
		local function noise() return "AAA", "BBB", "CCC" end
		local function arm()
			return debug.sethook(function(ev) events[#events + 1] = ev; noise() end, "r")
		end
		arm()
		local function four() return select(2, "p", "q", "r", "s") end
		local b = table.pack(four())
		debug.sethook()
		assert(events[1] == "return", "first event: " .. tostring(events[1]))
		assert(b.n == 3 and b[1] == "q" and b[2] == "r" and b[3] == "s",
			"select results: " .. table.concat(b, ",", 1, b.n))
	`
	runLuaWithDebug(t, source, "test_tailcall_hook_armed_by_native", provider)
}

// A count or line hook can never fire a return event, so a tail call into a
// native must behave exactly as it does with no hook at all.
func TestTailCallResultsUnderACountOnlyHook(t *testing.T) {
	provider := vm.NewDefaultDebugProvider()
	source := `
		local ticks = 0
		local function four() return select(2, "p", "q", "r", "s") end
		debug.sethook(function() ticks = ticks + 1 end, "", 1)
		local a = table.pack(four())
		debug.sethook()
		assert(a.n == 3 and a[1] == "q" and a[2] == "r" and a[3] == "s",
			"select results: " .. table.concat(a, ",", 1, a.n))
		assert(ticks > 0, "count hook never fired")
	`
	runLuaWithDebug(t, source, "test_tailcall_count_hook", provider)
}
