-- Bug: print() does not respect __tostring metamethod.
-- Lua 5.4's print() calls tostring() on each argument, which respects __tostring.
-- golua's print() calls valueToString() which does NOT check __tostring.

-- Test that print() invokes __tostring by checking for side effect
local called = false
local mt = {__tostring = function(self) called = true; return "custom" end}
local obj = setmetatable({}, mt)

-- tostring() already works (verify baseline)
assert(tostring(obj) == "custom", "tostring should respect __tostring")

-- print() should also invoke __tostring
called = false
print(obj)
assert(called, "print() should have invoked __tostring metamethod")

print("PASS")
