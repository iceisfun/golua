-- test_math_modf: math.modf edge cases

-- integer input
do
    local i, f = math.modf(23)
    assert(i == 23, string.format("modf(23) integer part: expected 23, got %s", tostring(i)))
    assert(f == 0, string.format("modf(23) fractional part: expected 0, got %s", tostring(f)))
end

-- positive float
do
    local i, f = math.modf(1.5)
    assert(i == 1, "modf(1.5) integer part")
    assert(f == 0.5, "modf(1.5) fractional part")
end

-- negative float
do
    local i, f = math.modf(-1.5)
    assert(i == -1, "modf(-1.5) integer part")
    assert(f == -0.5, "modf(-1.5) fractional part")
end

-- positive infinity
do
    local i, f = math.modf(1/0)
    assert(i == math.huge, string.format("modf(inf) integer part: expected inf, got %s", tostring(i)))
    assert(f == 0, string.format("modf(inf) fractional part: expected 0, got %s", tostring(f)))
end

-- negative infinity
do
    local i, f = math.modf(-1/0)
    assert(i == -math.huge, string.format("modf(-inf) integer part: expected -inf, got %s", tostring(i)))
    assert(f == 0, string.format("modf(-inf) fractional part: expected 0, got %s", tostring(f)))
end

-- no args should error
assert(not pcall(math.modf), "modf() no args")

-- table arg should error
assert(not pcall(math.modf, {}), "modf({}) should error")
