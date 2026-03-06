-- string.dump: serialize a function to binary bytecode
assert(string.dump ~= nil, "string.dump should exist")

local function f(x) return x * x end
local dumped = string.dump(f)
assert(type(dumped) == "string", "dump should return string")
assert(#dumped > 0, "dump should return non-empty string")

-- Round-trip via load
local f2 = load(dumped)
assert(f2 ~= nil, "load should accept dumped bytecode")
assert(f2(5) == 25, "loaded function should work")
