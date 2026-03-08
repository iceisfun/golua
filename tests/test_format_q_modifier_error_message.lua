-- Bug: error message for %q with modifiers differs from Lua 5.4
-- GoLua: "cannot have modifiers with '%q'"
-- Lua 5.4: "specifier '%q' cannot have modifiers"

local ok, err = pcall(string.format, "%10q", "hi")
assert(ok == false)
assert(err:find("specifier '%%q' cannot have modifiers"),
    "expected \"specifier '%q' cannot have modifiers\", got: " .. err)

print("PASSED")
