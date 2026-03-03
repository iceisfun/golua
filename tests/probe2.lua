-- probe2.lua

local function test(name, fn)
    print("TEST " .. name .. ": ")
    local ok, err = pcall(fn)
    if ok then
        print("PASS")
    else
        print("FAIL - " .. tostring(err))
    end
end

test("comparison lessThan with metamethods", function()
    local mt = {
        __lt = function(a, b) return a.val < b.val end
    }
    local a = setmetatable({val=1}, mt)
    local b = setmetatable({val=2}, mt)
    assert(a < b)
    assert(not (b < a))
end)

test("comparison lessEqual with metamethods", function()
    local mt1 = {
        __le = function(a, b) return a.val <= b.val end
    }
    local a1 = setmetatable({val=1}, mt1)
    local b1 = setmetatable({val=2}, mt1)
    assert(a1 <= b1)

    local mt2 = {
        __lt = function(a, b) return a.val < b.val end
    }
    local a2 = setmetatable({val=1}, mt2)
    local b2 = setmetatable({val=2}, mt2)
    -- Lua 5.4: if no __le, a <= b is converted to not (b < a)
    assert(a2 <= b2)
end)

test("string concat metamethod", function()
    local mt = {
        __concat = function(a, b)
            local av = type(a) == "table" and a.val or a
            local bv = type(b) == "table" and b.val or b
            return av .. bv
        end
    }
    local a = setmetatable({val="A"}, mt)
    local b = setmetatable({val="B"}, mt)
    assert(a .. b == "AB")
    assert(a .. "C" == "AC")
end)

test("string.gsub replacement function table", function()
    local t = {a = "A", b = "B"}
    assert(string.gsub("abc", ".", t) == "ABc")
end)
