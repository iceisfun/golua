-- Test: string.format rejects malformed specifiers that Lua 5.4 rejects

local function check_error(fmt_str, arg, expected_pattern)
    local ok, err = pcall(string.format, fmt_str, arg)
    assert(not ok, "expected error for: " .. fmt_str)
    assert(string.find(err, "invalid conversion specification:", 1, true),
           "expected 'invalid conversion specification:' in: " .. tostring(err))
end

-- Flags after width
check_error("%10-d", 42)
check_error("%10+d", 42)

-- Negative precision
check_error("%.-1f", 42.0)

-- Double dot
check_error("%..1d", 42)

-- Double precision
check_error("%.5.5f", 42.0)

-- Valid specifiers should still work
assert(string.format("%10d", 42) == "        42")
assert(string.format("%-10d", 42) == "42        ")
assert(string.format("%+d", 42) == "+42")
assert(string.format("%.5f", 42.0) == "42.00000")
assert(string.format("%#x", 42) == "0x2a")

print("OK")
