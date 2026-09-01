package golua_test

import (
	"fmt"
	"testing"

	"github.com/iceisfun/golua/vm"
)

// metaChainHelpers is prepended to the sources below. chain(n, tail) builds n
// nested tables whose __index ends at tail, schain(n, tail) the same for
// __newindex, and swap installs a string-metatable field around one call so the
// string methods used to check the result are still reachable afterwards.
const metaChainHelpers = `
local function chain(n, tail)
  local t = tail
  for i = 1, n do t = setmetatable({}, {__index = t}) end
  return t
end
local function schain(n, tail)
  local t = tail
  for i = 1, n do t = setmetatable({}, {__newindex = t}) end
  return t
end
local function swap(field, value, f)
  local smeta = getmetatable("")
  local saved = smeta[field]
  smeta[field] = value
  local ok, err = pcall(f)
  smeta[field] = saved
  return ok, err
end
local function tooLong(what, err)
  local msg = "'__" .. what .. "' chain too long; possible loop"
  return (tostring(err):find(msg, 1, true)) ~= nil
end
`

// A cycle that alternates between a table and a non-table __index value is a
// cycle like any other: it must raise the chain-too-long error and leave the VM
// usable, not walk the chain forever.
func TestIndexChainThroughNonTableValueTerminates(t *testing.T) {
	got := runLuaCapture(t, metaChainHelpers+`
local T = setmetatable({}, {__index = "s"})
local ok, err = swap("__index", T, function() return T.x end)
print(ok, tooLong("index", err))
print("survived")`)
	if got != "false\ttrue\nsurvived" {
		t.Fatalf("alternating __index cycle not contained: got %q", got)
	}
}

// The store path has the same cycle shape and the same requirement.
func TestNewIndexChainThroughNonTableValueTerminates(t *testing.T) {
	got := runLuaCapture(t, metaChainHelpers+`
local T = setmetatable({}, {__newindex = "s"})
local ok, err = swap("__newindex", T, function() T.x = 1 end)
print(ok, tooLong("newindex", err))
print("survived")`)
	if got != "false\ttrue\nsurvived" {
		t.Fatalf("alternating __newindex cycle not contained: got %q", got)
	}
}

// A chain that crosses a non-table __index value keeps draining the same depth
// budget across the hop; the two halves must not each be given a fresh one.
func TestIndexChainBudgetSpansNonTableHop(t *testing.T) {
	links := vm.DefaultMaxMetaDepth / 2
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local n = %d
local head = chain(n, setmetatable({}, {__index = "s"}))
local ok, err = swap("__index", chain(n, {x = "FOUND"}), function() return head.x end)
print(ok, tooLong("index", err))`, links))
	if got != "false\ttrue" {
		t.Fatalf("chain over the limit resolved instead of raising: got %q", got)
	}
}

func TestNewIndexChainBudgetSpansNonTableHop(t *testing.T) {
	links := vm.DefaultMaxMetaDepth / 2
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local n = %d
local sink = {}
local head = schain(n, setmetatable({}, {__newindex = "s"}))
local ok, err = swap("__newindex", schain(n, sink), function() head.k = 1 end)
print(ok, tooLong("newindex", err), sink.k)`, links))
	if got != "false\ttrue\tnil" {
		t.Fatalf("chain over the limit stored instead of raising: got %q", got)
	}
}

// Integer keys take their own specialised lookup path and share the budget too.
func TestIndexChainBudgetSpansNonTableHopIntegerKey(t *testing.T) {
	links := vm.DefaultMaxMetaDepth / 2
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local n = %d
local head = chain(n, setmetatable({}, {__index = "s"}))
local ok, err = swap("__index", chain(n, {[7] = "FOUND"}), function() return head[7] end)
print(ok, tooLong("index", err))`, links))
	if got != "false\ttrue" {
		t.Fatalf("integer-key chain over the limit resolved instead of raising: got %q", got)
	}
}

// A chain that stays under the limit must still resolve, including one that
// crosses a non-table __index value.
func TestIndexChainUnderLimitResolves(t *testing.T) {
	links := vm.DefaultMaxMetaDepth / 4
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local n = %d
local head = chain(n, setmetatable({}, {__index = "s"}))
print(swap("__index", chain(n, {x = "FOUND"}), function() return head.x end))
print(pcall(function() return chain(%d, {x = "DEEP"}).x end))`, links, vm.DefaultMaxMetaDepth))
	if got != "true\tFOUND\ntrue\tDEEP" {
		t.Fatalf("legal chain under the limit failed: got %q", got)
	}
}

func TestNewIndexChainUnderLimitResolves(t *testing.T) {
	links := vm.DefaultMaxMetaDepth / 4
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local n = %d
local sink = {}
local head = schain(n, setmetatable({}, {__newindex = "s"}))
print(swap("__newindex", schain(n, sink), function() head.k = 1 end), sink.k)`, links))
	if got != "true\t1" {
		t.Fatalf("legal __newindex chain under the limit failed: got %q", got)
	}
}

// A plain table-to-table cycle must keep being reported.
func TestIndexChainTableCycleStillDetected(t *testing.T) {
	got := runLuaCapture(t, metaChainHelpers+`
local a, b = {}, {}
setmetatable(a, {__index = b})
setmetatable(b, {__index = a})
local ok, err = pcall(function() return a.x end)
print(ok, tooLong("index", err))
local c, d = {}, {}
setmetatable(c, {__newindex = d})
setmetatable(d, {__newindex = c})
local ok2, err2 = pcall(function() c.x = 1 end)
print(ok2, tooLong("newindex", err2))`)
	if got != "false\ttrue\nfalse\ttrue" {
		t.Fatalf("plain metatable cycle no longer detected: got %q", got)
	}
}

// Indexing a non-table value walks the same shared budget: a chain reached
// through the string metatable must not restart it.
func TestIndexOnStringValueSharesChainBudget(t *testing.T) {
	got := runLuaCapture(t, metaChainHelpers+fmt.Sprintf(`
local ok, err = swap("__index", chain(%d, {x = "FOUND"}), function() return ("z").x end)
print(ok, tooLong("index", err))
print(swap("__index", chain(%d, {x = "FOUND"}), function() return ("z").x end))`,
		vm.DefaultMaxMetaDepth, vm.DefaultMaxMetaDepth-1))
	if got != "false\ttrue\ntrue\tFOUND" {
		t.Fatalf("string-value index chain limit off: got %q", got)
	}
}
