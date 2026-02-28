-- BUG: warn function not implemented
-- In Lua 5.4, warn() is a global function for emitting warnings.
-- GoLua doesn't provide it at all (warn == nil).

assert(type(warn) == "function", "warn should be a function, got: " .. type(warn))

-- Basic warn call should not error
local ok, err = pcall(warn, "test warning")
assert(ok, "warn('test warning') should not error: " .. tostring(err))

-- warn("@off") / warn("@on") control warning output
local ok2 = pcall(warn, "@off")
assert(ok2, "warn('@off') should not error")
local ok3 = pcall(warn, "@on")
assert(ok3, "warn('@on') should not error")
