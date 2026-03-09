-- Test _ENV override
-- When 'x' is an upvalue from an enclosing scope, it is resolved as an
-- upvalue, not through _ENV.  Only names that are NOT locals/upvalues
-- are looked up via _ENV (they are "globals" in Lua 5.4 semantics).

local x = 1

local f = function()
    local _ENV = { x = 2 }
    return x  -- 'x' is an upvalue (captures outer local), not _ENV["x"]
end

assert(f() == 1)   -- upvalue takes precedence over _ENV
assert(x == 1)

-- Verify that _ENV is used for names that are NOT upvalues
local g = function()
    local _ENV = { y = 42 }
    return y  -- 'y' is not a local/upvalue, resolved via _ENV
end

assert(g() == 42)
