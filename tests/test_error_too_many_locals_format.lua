-- Test: "too many local variables" error should include line, context, and near token
-- Bug: GoLua omits line number, "in main function"/"in function at line N", and "near '='"
-- Lua 5.4: [string "..."]:201: too many local variables (limit is 200) in main function near '='
-- GoLua:   [string "..."]: too many local variables (limit is 200)

-- Test in main function
local code = ""
for i = 1, 201 do
    code = code .. "local v" .. i .. " = " .. i .. "\n"
end
local _, err = load(code)
assert(err:find("in main function"), "missing 'in main function': " .. tostring(err))
assert(err:find("near '='"), "missing near '=': " .. tostring(err))
-- Check line number is present (should be :201:)
assert(err:find(":201:"), "missing line number: " .. tostring(err))

-- Test in named function
code = "local function f()\n"
for i = 1, 201 do
    code = code .. "local v" .. i .. " = " .. i .. "\n"
end
code = code .. "end"
_, err = load(code)
assert(err:find("in function at line 1"), "missing 'in function at line 1': " .. tostring(err))

print("PASS")
