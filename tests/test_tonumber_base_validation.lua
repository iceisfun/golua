-- test_tonumber_base_validation: invalid base should raise argument errors

local function expect_error(expr)
    local ok, err = pcall(expr)
    assert(ok == false, "expected error")
    assert(type(err) == "string" or type(err) == "userdata" or type(err) == "table", "expected non-nil error object")
end

-- Lua expects base in [2,36]
expect_error(function() tonumber("10", 1) end)
expect_error(function() tonumber("10", 0) end)
expect_error(function() tonumber("10", 37) end)

-- Base must be an integer
expect_error(function() tonumber("10", 10.5) end)
expect_error(function() tonumber("10", "x") end)

-- Adjacent valid base behavior
assert(tonumber("101", 2) == 5, "binary conversion should work")
assert(tonumber("z", 36) == 35, "base-36 conversion should work")
assert(tonumber("2", 2) == nil, "invalid digit for base should return nil")
assert(tonumber("-10", 16) == -16, "signed conversion should work")
assert(tonumber("  ff  ", 16) == 255, "base conversion should ignore surrounding spaces")
assert(tonumber("10", "10") == 10, "numeric-string base should be accepted")

-- With explicit base, non-string first arg should error
expect_error(function() tonumber(10, 2) end)

-- Control case (base-less): should still work
assert(tonumber("0xff") == 255, "base-less hex conversion should work")
