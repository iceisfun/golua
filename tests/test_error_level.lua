-- BROKEN: error() ignores level argument
-- In Lua 5.4, error(msg, level) prepends source location to string messages:
--   level 0: no location info
--   level 1: location where error() was called (default)
--   level 2: caller of the function that called error()

-- Level 1 should add location
local ok1, e1 = pcall(function() error("oops", 1) end)
assert(type(e1) == "string", "error message should be a string")
assert(e1:find(":.*: oops"), "level 1 should prepend source:line to message, got: " .. e1)

-- Level 0 should NOT add location
local ok0, e0 = pcall(function() error("raw", 0) end)
assert(e0 == "raw", "level 0 should not modify message, got: " .. tostring(e0))

-- Level 2 should blame caller
local function thrower() error("inner", 2) end
local function caller() thrower() end
local ok2, e2 = pcall(caller)
-- The location in e2 should point to the line calling thrower(), not the error() line
assert(type(e2) == "string" and e2:find(":.*: inner"), "level 2 should blame caller")
