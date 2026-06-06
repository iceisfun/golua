-- Lua 5.4: string.dump(f) serializes function f to a binary string.

-- string.dump should exist
assert(type(string.dump) == "function", "string.dump should be a function, got " .. type(string.dump))

-- Basic dump and reload
local f = function(x) return x + 1 end
local dumped = string.dump(f)
assert(type(dumped) == "string", "string.dump should return a string")
assert(#dumped > 0, "dumped string should be non-empty")

-- Guard: dump of function with upvalues should work (upvalues become nil on reload)
local upval = 10
local g = function() return upval end
local ok, result = pcall(string.dump, g)
assert(ok, "string.dump with upvalues should not error")

-- Guard: non-function argument should error
local ok2, e2 = pcall(string.dump, 42)
assert(not ok2, "string.dump(42) should error")
local ok3, e3 = pcall(string.dump, "hello")
assert(not ok3, "string.dump('hello') should error")
