-- String arithmetic metamethod falls back to other operand's metamethod
-- when the string can't be coerced to number

local mt = {
    __add = function(a,b) return "add" end,
    __sub = function(a,b) return "sub" end,
    __mul = function(a,b) return "mul" end,
    __div = function(a,b) return "div" end,
    __mod = function(a,b) return "mod" end,
    __pow = function(a,b) return "pow" end,
    __idiv = function(a,b) return "idiv" end,
}
local t = setmetatable({}, mt)

-- String on left, table with metamethod on right
assert("x" + t == "add")
assert("x" - t == "sub")
assert("x" * t == "mul")
assert("x" / t == "div")
assert("x" % t == "mod")
assert("x" ^ t == "pow")
assert("x" // t == "idiv")

-- Table on left, string on right
assert(t + "x" == "add")
assert(t - "x" == "sub")

print("PASS")
