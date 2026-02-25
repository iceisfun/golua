-- test_tonumber_default_edge_cases: Lua 5.4 default-base conversion strictness

-- Signed hex forms are valid in default mode
assert(tonumber("-0x10") == -16, "tonumber('-0x10') should be -16")
assert(tonumber("+0x10") == 16, "tonumber('+0x10') should be 16")
assert(tonumber("-0x1p4") == -16.0, "tonumber('-0x1p4') should be -16.0")
assert(tonumber("+0x1p4") == 16.0, "tonumber('+0x1p4') should be 16.0")

-- Lua 5.4 default tonumber should not accept inf/nan textual forms
assert(tonumber("inf") == nil, "tonumber('inf') should be nil")
assert(tonumber("nan") == nil, "tonumber('nan') should be nil")

-- Lua 5.4 default tonumber should not accept underscore separators
assert(tonumber("1_000") == nil, "tonumber('1_000') should be nil")

-- Numeric overflow can still produce infinities in default mode.
assert(tonumber("1e309") == math.huge, "tonumber('1e309') should be inf")
