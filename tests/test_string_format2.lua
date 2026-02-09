-- test_string_format2: string.format specifiers and edge cases

-- %s
assert(string.format("%s=%f", "pi", 3.14) == "pi=3.140000", "format %s %f")

-- %s with nil, true, false
assert(string.format("-%s-%s-%s", nil, true, false) == "-nil-true-false", "format %s special")

-- %%
assert(string.format("%%") == "%", "format percent-escape")

-- %q string quoting
do
    local r = string.format("%q", '"hello"\t123')
    assert(r ~= nil and #r > 0, "format %q basic")
end

-- %d with width/padding
assert(string.format("%d", 10) == "10", "format %d basic")
assert(string.format("%05d", 10) == "00010", "format %d zero-pad")

-- %f with precision
assert(string.format("%.2f", 3.14) == "3.14", "format %f precision")

-- %c char
assert(string.format("%c", 65) == "A", "format %c")

-- %x hex
assert(string.format("%x", 255) == "ff", "format %x")

-- %e scientific
do
    local r = string.format("%e", 1.5)
    assert(r ~= nil and r:find("e") ~= nil, "format %e")
end

-- Too many values is OK
assert(string.format("%s", 1, 2, 3) == "1", "format extra values")

-- No args should error
assert(not pcall(string.format), "format no args")
