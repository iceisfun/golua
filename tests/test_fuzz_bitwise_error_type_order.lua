-- Regression: bitwise ops must evaluate type errors in the same order as Lua 5.4/5.5.
-- When LHS is a non-integer float and RHS is a non-numeric type, the error must
-- report the RHS type mismatch ("bitwise operation on a <type> value"), not the
-- LHS coercion failure ("number has no integer representation"). Previously golua
-- eagerly coerced the LHS before classifying the RHS, producing the wrong error.

local function check(fn, expectedSubstr, label)
    local ok, err = pcall(fn)
    assert(not ok, label .. ": expected error, got success")
    assert(type(err) == "string", label .. ": error must be a string, got " .. type(err))
    assert(string.find(err, expectedSubstr, 1, true),
        label .. ": expected substring " .. expectedSubstr .. ", got: " .. tostring(err))
end

-- BAND: non-integer float LHS, non-number RHS -> RHS type error wins.
check(function() return 1.5 & nil end,      "bitwise operation on a nil value",    "1.5 & nil")
check(function() return 1.5 & {} end,       "bitwise operation on a table value",  "1.5 & {}")
check(function() return 1.5 & true end,     "bitwise operation on a boolean value","1.5 & true")

-- BOR
check(function() return 1.5 | nil end,      "bitwise operation on a nil value",    "1.5 | nil")
check(function() return 1.5 | "abc" end,    "bitwise operation on a string value", "1.5 | 'abc'")
check(function() return 1.5 | {} end,       "bitwise operation on a table value",  "1.5 | {}")

-- BXOR
check(function() return 1.5 ~ nil end,      "bitwise operation on a nil value",    "1.5 ~ nil")
check(function() return 1.5 ~ "abc" end,    "bitwise operation on a string value", "1.5 ~ 'abc'")
check(function() return 1.5 ~ {} end,       "bitwise operation on a table value",  "1.5 ~ {}")

-- SHL
check(function() return 1.5 << nil end,     "bitwise operation on a nil value",    "1.5 << nil")
check(function() return 1.5 << "abc" end,   "bitwise operation on a string value", "1.5 << 'abc'")
check(function() return 1.5 << {} end,      "bitwise operation on a table value",  "1.5 << {}")

-- SHR
check(function() return 1.5 >> nil end,     "bitwise operation on a nil value",    "1.5 >> nil")
check(function() return 1.5 >> "abc" end,   "bitwise operation on a string value", "1.5 >> 'abc'")
check(function() return 1.5 >> {} end,      "bitwise operation on a table value",  "1.5 >> {}")

-- Mirror form: non-number LHS, non-integer float RHS -> LHS type error (already correct).
check(function() return nil & 1.5 end,      "bitwise operation on a nil value",    "nil & 1.5")
check(function() return "abc" | 1.5 end,    "bitwise operation on a string value", "'abc' | 1.5")
check(function() return {} ~ 1.5 end,       "bitwise operation on a table value",  "{} ~ 1.5")

-- Both operands are numbers but at least one is non-integer -> "no integer representation".
check(function() return 1.5 & 2 end,        "has no integer representation",       "1.5 & 2")
check(function() return 2 & 1.5 end,        "has no integer representation",       "2 & 1.5")
check(function() return 1.5 & 2.5 end,      "has no integer representation",       "1.5 & 2.5")
check(function() return 1.5 << 2 end,       "has no integer representation",       "1.5 << 2")

-- Valid cases must still work (float with integer representation coerces).
assert((1.0 & 3) == 1, "1.0 & 3 should be 1")
assert((1.0 | 2.0) == 3, "1.0 | 2.0 should be 3")
assert((7 ~ 2.0) == 5, "7 ~ 2.0 should be 5")
assert((1.0 << 3) == 8, "1.0 << 3 should be 8")
assert((16.0 >> 2) == 4, "16.0 >> 2 should be 4")

print("OK")
