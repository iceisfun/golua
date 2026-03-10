-- test_string_format: string.format specifiers and edge cases

-- %s/%d/%i basics
assert(string.format("%s=%f", "pi", 3.14) == "pi=3.140000", "format %s %f")
assert(string.format("-%s-%s-%s", nil, true, false) == "-nil-true-false", "format %s special")
assert(string.format("%%") == "%", "format percent-escape")
assert(string.format("%d", 10) == "10", "format %d basic")
assert(string.format("%05d", 10) == "00010", "format %d zero-pad")
assert(string.format("%i", -12) == "-12", "format %i")
assert(string.format("-%u-", 55) == "-55-", "format %u")
assert(string.format("%s", 1, 2, 3) == "1", "format extra values")

-- %f/%e
assert(string.format("%.2f", 3.14) == "3.14", "format %f precision")
do
    local r = string.format("%e", 1.5)
    assert(r ~= nil and r:find("e") ~= nil, "format %e")
end

-- %g (6 significant digits by default)
assert(string.format("%g", 12345.6789) == "12345.7",
  "expected '12345.7', got '" .. string.format("%g", 12345.6789) .. "'")
assert(string.format("%g", 123456.789) == "123457",
  "expected '123457', got '" .. string.format("%g", 123456.789) .. "'")
assert(string.format("%g", 1234567.89) == "1.23457e+06",
  "expected '1.23457e+06', got '" .. string.format("%g", 1234567.89) .. "'")
assert(string.format("%G", 12345.6789) == "12345.7",
  "expected '12345.7', got '" .. string.format("%G", 12345.6789) .. "'")
assert(string.format("%.10g", 12345.6789) == "12345.6789",
  "explicit precision should work")
assert(string.format("%g", 42) == "42", "simple integer should format as '42'")
assert(string.format("%g", 0.5) == "0.5", "simple float should format as '0.5'")

-- %a (hex float)
assert(string.format("%a", 1.5) == "0x1.8p+0", "%a for 1.5")
assert(string.format("%A", 1.5) == "0X1.8P+0", "%A for 1.5")
assert(string.format("%a", -16.0) == "-0x1p+4", "%a for -16")
do
    local basic = string.format("%a", 1.5)
    assert(basic:find("0x") and basic:find("p"),
      "basic %a should produce hex float, got: " .. basic)
end
do
    local prec2 = string.format("%.2a", 1.5)
    assert(prec2 == "0x1.80p+0",
      "expected '0x1.80p+0', got '" .. prec2 .. "'")
end
do
    local prec4 = string.format("%.4a", math.pi)
    assert(prec4 == "0x1.9220p+1",
      "expected '0x1.9220p+1', got '" .. prec4 .. "'")
end
do
    local prec0 = string.format("%.0a", 1.5)
    assert(prec0 == "0x2p+0",
      "expected '0x2p+0', got '" .. prec0 .. "'")
end
do
    local upper = string.format("%A", 1.5)
    assert(upper:find("0X") and upper:find("P"),
      "basic %A should produce uppercase hex float, got: " .. upper)
end
do
    local ok, err = pcall(function() return string.format("%a", "nope") end)
    assert(ok == false, "%a with non-number should error")
    assert(type(err) == "string" and err:find("number expected"),
           "unexpected %a error: " .. tostring(err))
end

-- %q (quoted)
do
    local r = string.format("%q", '"hello"\t123')
    assert(r ~= nil and #r > 0, "format %q basic")
end
do
    -- %q of a float must roundtrip as a float, not an integer
    local s = string.format("%q", 1.0)
    local f = load("return " .. s)
    assert(f, "load of %q float should work: " .. s)
    local val = f()
    assert(math.type(val) == "float",
      "%q of 1.0 should roundtrip as float, got " .. math.type(val) .. " from: " .. s)
    assert(val == 1.0, "%q of 1.0 should roundtrip to 1.0")
end
do
    -- %q of pi should roundtrip exactly
    local s2 = string.format("%q", math.pi)
    local f2 = load("return " .. s2)
    assert(f2, "load of %q pi should work: " .. s2)
    assert(f2() == math.pi, "%q of pi should roundtrip exactly")
end
do
    -- %q with math.mininteger
    local s3 = string.format("%q", math.mininteger)
    local f3 = load("return " .. s3)
    assert(f3, "load of %q mininteger should work: " .. s3)
    local val3 = f3()
    assert(val3 == math.mininteger,
      "%q of mininteger should roundtrip, got: " .. tostring(val3))
    assert(math.type(val3) == "integer",
      "%q of mininteger should roundtrip as integer, got " .. math.type(val3))
end
assert(string.format("[%q]", nil) == "[nil]", "format %q nil")
assert(string.format("%q", false) == "false", "format %q false")
assert(string.format("%q,%q", math.huge, -math.huge) == "1e9999,-1e9999", "format %q inf")
assert(string.format("%q", 0/0) == "(0/0)", "format %q nan")
do
    local ok, err = pcall(string.format, "%q", {})
    assert(not ok, "%q of table should error")
    assert(tostring(err):find("no literal form") or tostring(err):find("has no literal"),
      "%q table error should mention 'no literal form', got: " .. tostring(err))
end
do
    local ok2, err2 = pcall(string.format, "%q", print)
    assert(not ok2, "%q of function should error")
end

-- %c
assert(string.format("%c", 65) == "A", "format %c")
assert(string.format("%c", 65.0) == "A", "%c with integer float should work")
do
    local ok1, err1 = pcall(function() return string.format("%c", 65.1) end)
    assert(ok1 == false, "%c with fractional float should error")
    assert(type(err1) == "string" and err1:find("number has no integer representation"),
           "unexpected %c fractional error: " .. tostring(err1))
end
do
    local ok2, err2 = pcall(function() return string.format("%c", "65.1") end)
    assert(ok2 == false, "%c with fractional numeric string should error")
    assert(type(err2) == "string" and err2:find("number has no integer representation"),
           "unexpected %c fractional-string error: " .. tostring(err2))
end
assert(string.byte(string.format("%c", 255)) == 255, "%c 255 should produce byte 255")
assert(string.byte(string.format("%c", 256)) == 0, "%c 256 should wrap to byte 0")
assert(string.byte(string.format("%c", -1)) == 255, "%c -1 should wrap to byte 255")

-- %x/%X/%o
assert(string.format("%x", 255) == "ff", "format %x")
assert(string.format("txt%#5.5o%0#10.5X", 1, 123) == "txt00001   0X0007B",
  "precision should disable integer zero-padding")

-- %p pointer format
assert(string.format("%p", 1) == "(null)", "format %p int")
assert(string.format("%p", true) == "(null)", "format %p bool")
assert(string.format("%p", nil) == "(null)", "format %p nil")
do
    local t = {}
    local r = string.format("=%p=", t)
    assert(r:match("^=0x[0-9a-f]+="), "format %p table should give hex pointer")
end

-- Integer specs must reject non-integer numbers
do
    local function expect_no_int_repr(spec, value)
        local ok, err = pcall(function() return string.format(spec, value) end)
        assert(ok == false, spec .. " should reject value " .. tostring(value))
        assert(type(err) == "string" and err:find("number has no integer representation"),
               "unexpected error for " .. spec .. ": " .. tostring(err))
    end

    for _, spec in ipairs({"%d", "%i", "%u", "%x", "%X", "%o"}) do
        expect_no_int_repr(spec, 3.9)
        expect_no_int_repr(spec, "3.9")
    end

    assert(string.format("%d", 3.0) == "3", "%d should accept integer float")
    assert(string.format("%x", 15.0) == "f", "%x should accept integer float")
end

-- Special float formatting (inf/nan)
assert(string.format("%f", 1/0) == "inf", "%f +inf should be 'inf'")
assert(string.format("%+f", 1/0) == "+inf", "%+f +inf should be '+inf'")
assert(string.format("%8f", 1/0) == "     inf", "%8f +inf width mismatch")
assert(string.format("%-8f", 1/0) == "inf     ", "%-8f +inf width mismatch")
assert(string.format("%f", -1/0) == "-inf", "%f -inf should be '-inf'")
assert(string.format("%f", 0/0) == "-nan", "%f nan should be '-nan'")
assert(string.format("%E", 1/0) == "INF", "%E +inf should be 'INF'")
assert(string.format("%G", -1/0) == "-INF", "%G -inf should be '-INF'")
assert(string.format("%A", 0/0) == "-NAN", "%A nan should be '-NAN'")
do
    local ok, err = pcall(function() return string.format("%F", 1.0) end)
    assert(ok == false, "%F should be invalid in Lua 5.4")
    assert(type(err) == "string" and err:find("invalid conversion") and err:find("%%F"),
           "unexpected %F error: " .. tostring(err))
end

-- Invalid format specifiers should error
do
    local ok1, err1 = pcall(string.format, "%z", 42)
    assert(not ok1, "%z should be invalid format")
    assert(tostring(err1):find("invalid"),
      "error should mention 'invalid', got: " .. tostring(err1))
end
do
    local ok2, err2 = pcall(string.format, "%r", 42)
    assert(not ok2, "%r should be invalid format")
end
do
    local ok3, err3 = pcall(string.format, "%n", 42)
    assert(not ok3, "%n should be invalid format")
end

-- Error cases
assert(not pcall(string.format), "format no args")
assert(not pcall(string.format, "%s %d", 1), "format not enough values should error")
