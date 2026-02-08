-- test_math_randomseed.lua
-- math.randomseed should support 0/1/2-argument forms and return two numbers per Lua 5.4.

local function expect_two_numbers(...)
    local ok, a, b = ...
    assert(ok, select(2, ...))
    assert(type(a) == "number" and type(b) == "number", "expected math.randomseed to return two numbers")
end

-- No-arg form should succeed (seed from entropy) and return two numbers
expect_two_numbers(pcall(math.randomseed))

-- One-arg form should succeed and return two numbers
expect_two_numbers(pcall(math.randomseed, 12345))

-- Two-arg form should succeed and return two numbers
expect_two_numbers(pcall(math.randomseed, 111, 222))
