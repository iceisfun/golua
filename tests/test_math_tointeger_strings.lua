-- test_math_tointeger_strings.lua
-- math.tointeger must not coerce strings per Lua 5.4; it should return nil.

local vals = {
    "5",
    "  42  ",
    "0x10",
    "3.14",
    "1e3",
}

for _, v in ipairs(vals) do
    assert(math.tointeger(v) == nil, string.format("math.tointeger(%q) should be nil", v))
end

-- Behavior should still work for actual numbers
assert(math.tointeger(5.0) == 5, "numeric input should still convert")
