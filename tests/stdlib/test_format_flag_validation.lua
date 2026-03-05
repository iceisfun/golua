-- string.format flag validation: + and space flags are only valid for
-- d, i, f, e, E, g, G, a, A conversions. They must be rejected for
-- o, x, X, u, s, c, p.

local function expect_error(fmt_str, val, label)
    local ok, err = pcall(string.format, fmt_str, val)
    assert(not ok, label .. " should error")
    assert(tostring(err):find("invalid conversion"), label .. " wrong error: " .. tostring(err))
end

-- + flag rejected for o, x, X, u, s
expect_error("%+x", 0, "%+x")
expect_error("%+X", 0, "%+X")
expect_error("%+o", 0, "%+o")
expect_error("%+u", 0, "%+u")
expect_error("%+s", "hi", "%+s")

-- space flag rejected for o, x, X, u, s
expect_error("% x", 0, "% x")
expect_error("% X", 0, "% X")
expect_error("% o", 0, "% o")
expect_error("% u", 0, "% u")
expect_error("% s", "hi", "% s")

-- + and space flags ARE valid for d, i
assert(string.format("%+d", 42) == "+42")
assert(string.format("% d", 42) == " 42")
assert(string.format("%+i", 42) == "+42")
assert(string.format("% i", 42) == " 42")

-- + and space flags ARE valid for f, e, E, g, G
assert(string.format("%+f", 1.0):sub(1,1) == "+")
assert(string.format("% f", 1.0):sub(1,1) == " ")
assert(string.format("%+e", 1.0):sub(1,1) == "+")
assert(string.format("%+g", 1.0):sub(1,1) == "+")

-- # flag rejected for d, i, u (but valid for o, x, X)
expect_error("%#d", 0, "%#d")
expect_error("%#i", 0, "%#i")
expect_error("%#u", 0, "%#u")

-- # flag IS valid for o, x, X
assert(string.format("%#x", 1) == "0x1")
assert(string.format("%#o", 1) == "01")

print("OK")
