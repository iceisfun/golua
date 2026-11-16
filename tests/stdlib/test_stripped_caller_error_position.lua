-- native errors raised from a stripped Lua caller carry no position.

local g = load(string.dump(function() string.rep() end, true))
local ok4, e4 = pcall(g)
assert(not ok4 and e4 == "bad argument #1 to 'rep' (string expected, got no value)", e4)

print("OK")
