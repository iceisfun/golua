-- "too many upvalues" should be reported correctly
-- When a function captures more than 255 upvalues, the error should say
-- "too many upvalues" and include "in function at line N"

do
    local parts = {}
    -- Outer function with 199 locals
    parts[#parts+1] = "local function outer()"
    for i = 1, 199 do
        parts[#parts+1] = string.format("local v%d = %d", i, i)
    end
    -- Middle function with 56 more locals
    parts[#parts+1] = "local function middle()"
    for i = 200, 255 do
        parts[#parts+1] = string.format("local w%d = %d", i, i)
    end
    -- Inner function captures all 255 from outer + middle + _ENV = 256 upvalues
    -- Use a table constructor to avoid deep expression nesting
    parts[#parts+1] = "local function inner()"
    parts[#parts+1] = "return {print,"  -- forces _ENV upvalue
    local refs = {}
    for i = 1, 199 do
        refs[#refs+1] = string.format("v%d", i)
    end
    for i = 200, 255 do
        refs[#refs+1] = string.format("w%d", i)
    end
    parts[#parts+1] = table.concat(refs, ",")
    parts[#parts+1] = "}"
    parts[#parts+1] = "end"  -- inner
    parts[#parts+1] = "end"  -- middle
    parts[#parts+1] = "end"  -- outer

    local code = table.concat(parts, "\n")
    local f, err = load(code)
    print(f)
    --> =nil
    -- Error should mention "upvalues" not "registers"
    print(err:match("too many upvalues") ~= nil)
    --> =true
    -- Should include "in function at line N"
    print(err)
    --> ~in function at line
end
