-- Lua 5.4+ compatibility tests

-- 1. Attributes and Scope
local closed = false
local function test_scope()
    local x <const> = 50
    local obj <close> = setmetatable({}, {
        __close = function() closed = true end
    })
    assert(x == 50)
end
test_scope()
assert(closed == true, "Attribute <close> failed to trigger __close")

-- 2. Coercion and Arithmetic
local sum = "10" + 5
assert(type(sum) == "number")
assert(sum == 15)

local floor_div = 10 // 3
assert(floor_div == 3)

local bitwise = 1 << 3
assert(bitwise == 8)

-- 3. Closures and Upvalues
local function power_factory(n)
    return function(x) return x ^ n end
end
local square = power_factory(2)
local cube = power_factory(3)
assert(square(4) == 16)
assert(cube(2) == 8)

-- 4. String Library
local s_start, s_end = string.find("hello world", "lo")
assert(s_start == 4 and s_end == 5)

local date = "2026-02-07"
local y, m, d = string.match(date, "(%d+)-(%d+)-(%d+)")
assert(y == "2026", "Year match failed")
assert(m == "02")
assert(d == "07")

local function upper(s) return string.upper(s) end
local shouted = string.gsub("hello", "%w", upper)
assert(shouted == "HELLO", "gsub function callback failed")

local lookup = { a = "1", b = "2" }
local replaced = string.gsub("a-b", "%w", lookup)
assert(replaced == "1-2", "gsub table lookup failed")

-- 5. Table Length & Move
local t = {1, 2, 3, 4, 5}
table.move(t, 1, 2, 6)
assert(t[6] == 1 and t[7] == 2)

local holey = {10, 20, nil, 40}
assert(type(#holey) == "number")

-- 6. Metatables
local meta = { __index = function(_, k) return "key_" .. k end }
local proxy = setmetatable({}, meta)
assert(proxy.test == "key_test")
