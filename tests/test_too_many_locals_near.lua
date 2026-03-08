-- Test that "too many local variables" reports near ')' or ',' not near parameter name

-- Generate a function with 201 parameters
local params = {}
for i = 0, 200 do
    params[#params+1] = "p" .. i
end
local code = "function f(" .. table.concat(params, ",") .. ") end"

local ok, err = load(code)
assert(not ok, "201 params should fail")
assert(err:find("too many local variables"), "should say 'too many local variables': " .. err)
-- Lua 5.4 says near ')' — the token after the last parameter
-- GoLua was incorrectly saying near 'p200'
assert(not err:find("near 'p%d+"), "should not say near parameter name: " .. err)

print("OK")
