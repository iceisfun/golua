-- probe3.lua

local function test(name, fn)
    print("TEST " .. name .. ": ")
    local ok, err = pcall(fn)
    if ok then
        print("PASS")
    else
        print("FAIL - " .. tostring(err))
    end
end

test("sort with invalid comparator", function()
    local t = {1, 2, 3}
    -- Invalid comparator (always returns true) should cause an error in Lua 5.4 
    -- if it violates the strict weak ordering requirement, or at least not crash the VM.
    table.sort(t, function(a, b) return true end)
end)

test("frontier pattern", function()
    local s = "the word"
    -- Match 'word' only if it's not preceded by a letter
    local m = string.match(s, "%f[%w]word%f[%W]")
    assert(m == "word")
end)

test("vararg in table constructor", function()
    local function f(...)
        return {1, ...}
    end
    local t = f(2, 3)
    assert(t[1] == 1 and t[2] == 2 and t[3] == 3)
end)

test("__newindex does not return value", function()
    local mt = {
        __newindex = function(t, k, v)
            rawset(t, k, v)
            return 42
        end
    }
    local t = setmetatable({}, mt)
    -- Assignment is a statement, not an expression, so it doesn't return in Lua.
    -- We just verify it executes without error.
    t.a = 1 
    assert(t.a == 1)
end)
