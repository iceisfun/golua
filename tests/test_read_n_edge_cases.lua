-- read("n") edge cases for number parsing from files.

-- Leading zeros should be decimal, not octal
do
    local f = io.tmpfile()
    f:write("010 042 0100")
    f:seek("set")
    local a, b, c = f:read("n"), f:read("n"), f:read("n")
    assert(a == 10, "010 should be 10 (decimal), got " .. tostring(a))
    assert(b == 42, "042 should be 42, got " .. tostring(b))
    assert(c == 100, "0100 should be 100, got " .. tostring(c))
    f:close()
end

-- Negative integer overflow should produce float
do
    local f = io.tmpfile()
    f:write("-9223372036854775809")
    f:seek("set")
    local v = f:read("n")
    assert(math.type(v) == "float",
        "negative overflow should be float, got " .. math.type(v) .. " = " .. tostring(v))
    f:close()
end

-- Very large exponent should produce inf
do
    local f = io.tmpfile()
    f:write("1e1000 -1e1000")
    f:seek("set")
    local a, b = f:read("n"), f:read("n")
    assert(a == math.huge, "1e1000 should be inf, got " .. tostring(a))
    assert(b == -math.huge, "-1e1000 should be -inf, got " .. tostring(b))
    f:close()
end

-- Hex float without p exponent (0x1.8 = 1.5)
do
    local f = io.tmpfile()
    f:write("0x1.8")
    f:seek("set")
    local v = f:read("n")
    assert(v == 1.5, "0x1.8 should be 1.5, got " .. tostring(v))
    assert(math.type(v) == "float", "0x1.8 should be float, got " .. math.type(v))
    f:close()
end

-- Hex float with p exponent should be float type
do
    local f = io.tmpfile()
    f:write("0xABp0")
    f:seek("set")
    local v = f:read("n")
    assert(v == 171.0, "0xABp0 should be 171.0, got " .. tostring(v))
    assert(math.type(v) == "float", "0xABp0 should be float, got " .. math.type(v))
    f:close()
end
