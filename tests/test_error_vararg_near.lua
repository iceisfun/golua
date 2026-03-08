-- Test: Vararg outside vararg function error should include "near '...'"
-- Bug: GoLua omits the "near '...'" suffix.
-- Lua 5.4: cannot use '...' outside a vararg function near '...'
-- GoLua:   cannot use '...' outside a vararg function

local _, err = load("local function f() return ... end")
assert(err:find("near '%.%.%.'"), "missing near '...': " .. tostring(err))

-- Also test in nested function
_, err = load("function f(...) local function g() return ... end end")
assert(err:find("near '%.%.%.'"), "nested missing near '...': " .. tostring(err))

print("PASS")
