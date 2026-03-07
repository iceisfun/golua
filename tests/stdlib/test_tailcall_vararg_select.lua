-- BUG: select(n, ...) corrupts varargs in tail-called functions
--
-- When a function is invoked via tail call (return f(args)), and the
-- callee uses select(n, ...) with a numeric index, the vararg storage
-- gets corrupted. Subsequent reads of ... (via select, print, etc.)
-- return wrong values or nil.
--
-- select('#', ...) does NOT trigger the corruption.
-- Non-tail calls work correctly.
-- Direct vararg reads (local a,b,c = ...) and {...} work correctly.

-- Test 1: select(1, ...) followed by select(2, ...) in tail-called function
local function f(...)
  local a = select(1, ...)
  local b = select(2, ...)
  return a, b
end
local function tc() return f(10, 20) end
local a, b = tc()
assert(a == 10 and b == 20,
  string.format("Test 1 failed: expected 10, 20 got %s, %s", tostring(a), tostring(b)))

-- Test 2: select(1, ...) then ... expansion
local function g(...)
  local _ = select(1, ...)
  return ...
end
local function tc2() return g(10, 20, 30) end
local x, y, z = tc2()
assert(x == 10 and y == 20 and z == 30,
  string.format("Test 2 failed: expected 10, 20, 30 got %s, %s, %s",
    tostring(x), tostring(y), tostring(z)))

-- Test 3: function with fixed params + vararg, tail-called
local function h(a, ...)
  local n = select('#', ...)
  local first = select(1, ...)
  return a, n, first
end
local function tc3() return h(10, 20, 30) end
local p, q, r = tc3()
assert(p == 10 and q == 2 and r == 20,
  string.format("Test 3 failed: expected 10, 2, 20 got %s, %s, %s",
    tostring(p), tostring(q), tostring(r)))

-- Test 4: method-style tail call
local obj = {}
function obj:m(...) return select(1, ...), select(2, ...) end
local function tc4() return obj:m(10, 20) end
local m, n = tc4()
assert(m == 10 and n == 20,
  string.format("Test 4 failed: expected 10, 20 got %s, %s", tostring(m), tostring(n)))

