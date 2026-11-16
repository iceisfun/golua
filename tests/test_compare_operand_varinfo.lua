-- Variable info in error messages must survive a comparison in the same
-- expression. A materialised comparison emits a JMP whose offset bits decode
-- to an A operand of 0; the backward scan that names the faulting register
-- used to take that JMP as the register's definition and drop the name.
-- Expected messages are the ones reference Lua 5.5 produces.

local b = 1
local c = "k"

local cases = {
    {function() local r = ('s2')(b == 1) return r end,
     "attempt to call a string value %(constant 's2'%)"},
    {function() local t = {} local r = t.x(b == 1) return r end,
     "attempt to call a nil value %(field 'x'%)"},
    {function() local t = {} local r = t:x(b == 1) return r end,
     "attempt to call a nil value %(method 'x'%)"},
    {function() return c >> (1 > 0) end,
     "attempt to perform bitwise operation on a string value %(upvalue 'c'%)"},
    {function() local t = {} return t.y + (b < 2 and 1 or 0) end,
     "attempt to perform arithmetic on a nil value %(field 'y'%)"},
}

for i, case in ipairs(cases) do
    local ok, err = pcall(case[1])
    assert(not ok, "case " .. i .. ": expected an error")
    assert(err:find(case[2]), "case " .. i .. ": got: " .. err)
end

print("OK")
