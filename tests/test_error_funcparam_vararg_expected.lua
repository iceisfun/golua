-- Test: Function parameter list error should say "<name> or '...' expected"
-- Bug: GoLua says "<name> expected" without the "or '...'" alternative.
-- Lua 5.4: <name> or '...' expected near <eof>
-- GoLua:   <name> expected near <eof>

local _, err = load("function f(\n\n")
assert(err:find("<name> or '%.%.%.' expected"),
    "missing 'or ...': got: " .. tostring(err))

print("PASS")
