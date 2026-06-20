-- Test: "too many local variables" error should include line, context, and near token.
-- Lua 5.5 reports the limit from adjustlocalvars, which runs after the whole
-- declaration is parsed, so the line and near-token are those of the token
-- following the offending statement (here EOF at line 202), not the '=' inside it:
--   [string "..."]:202: too many local variables (limit is 200) in main function near <eof>

-- Test in main function
local code = ""
for i = 1, 201 do
    code = code .. "local v" .. i .. " = " .. i .. "\n"
end
local _, err = load(code)
assert(err:find("in main function"), "missing 'in main function': " .. tostring(err))
assert(err:find("near <eof>"), "missing near <eof>: " .. tostring(err))
-- The limit is reported at the post-statement lookahead (EOF on line 202).
assert(err:find(":202:"), "missing line number: " .. tostring(err))

-- Test in named function
code = "local function f()\n"
for i = 1, 201 do
    code = code .. "local v" .. i .. " = " .. i .. "\n"
end
code = code .. "end"
_, err = load(code)
assert(err:find("in function at line 1"), "missing 'in function at line 1': " .. tostring(err))

print("PASS")
