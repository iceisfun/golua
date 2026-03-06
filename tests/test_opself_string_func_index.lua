-- OP_SELF on strings with function-based __index in metatable
-- Bug: OP_SELF only handled table __index, not function __index for strings

local mt = getmetatable("")
local orig = mt.__index

-- Replace with function __index
mt.__index = function(s, k)
    return orig[k]
end

-- Method call via : syntax should still work
assert(("hello"):upper() == "HELLO", "string:upper() failed with function __index")
assert(("hello"):sub(1,3) == "hel", "string:sub() failed with function __index")
assert(("hello"):len() == 5, "string:len() failed with function __index")
assert(("hello"):rep(2) == "hellohello", "string:rep() failed with function __index")

-- Restore
mt.__index = orig

print("PASS")
