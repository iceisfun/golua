-- test_string_format_integer_specs: integer specs must reject non-integer numbers

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

-- Integer-representable values are accepted
assert(string.format("%d", 3.0) == "3", "%d should accept integer float")
assert(string.format("%x", 15.0) == "f", "%x should accept integer float")
