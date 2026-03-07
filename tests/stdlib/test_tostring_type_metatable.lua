-- tostring() should respect __tostring on type metatables (set via debug.setmetatable).
-- Lua 5.4 Reference §6.1: tostring "checks whether the value has a __tostring
-- metamethod" — this applies to all types, not just tables/userdata.

-- Test 1: __tostring on number metatable
debug.setmetatable(0, {__tostring = function(v) return "num:" .. v end})
assert(tostring(42) == "num:42", "tostring(42) with number metatable, got: " .. tostring(42))
assert(tostring(3.14) == "num:3.14", "tostring(3.14) with number metatable, got: " .. tostring(3.14))
debug.setmetatable(0, nil) -- clean up

-- Test 2: __tostring on boolean metatable
debug.setmetatable(true, {__tostring = function(v) return "bool:" .. tostring(v) end})
-- Note: tostring(true) would recurse, so we test indirectly
local mt = debug.getmetatable(true)
assert(mt ~= nil, "boolean should have metatable")
assert(mt.__tostring ~= nil, "boolean metatable should have __tostring")
debug.setmetatable(true, nil) -- clean up

-- Test 3: __tostring on string metatable
debug.setmetatable("", {__tostring = function(v) return "str:" .. v end})
assert(tostring("hello") == "str:hello", "tostring('hello') with string metatable, got: " .. tostring("hello"))
debug.setmetatable("", nil) -- clean up

-- Test 4: print() should also use type metatables
debug.setmetatable(0, {__tostring = function(v) return "N" .. v end})
-- We can't easily test print output here, but tostring is the key
assert(tostring(100) == "N100", "tostring(100) should use metatable")
debug.setmetatable(0, nil) -- clean up

-- Test 5: without __tostring, default behavior
assert(tostring(42) == "42")
assert(tostring(true) == "true")
assert(tostring("hello") == "hello")
assert(tostring(nil) == "nil")


