-- Test: Unicode letters must be rejected as identifiers (ASCII-only)
-- Lua 5.4 only accepts [a-zA-Z_] for identifier start and [a-zA-Z0-9_] for continuation.

-- Unicode letter as identifier should fail to parse
local ok, err = load("local café = 1")
assert(not ok, "Unicode in identifier should be rejected")

-- Greek letter as identifier
local ok2, err2 = load("local α = 1")
assert(not ok2, "Greek letter identifier should be rejected")

-- CJK character as identifier
local ok3, err3 = load("local 中 = 1")
assert(not ok3, "CJK identifier should be rejected")

-- ASCII identifiers should still work fine
local ok4 = load("local abc_123 = 1")
assert(ok4, "ASCII identifier should be accepted")

-- Underscore-only identifier
local ok5 = load("local _ = 1")
assert(ok5, "Underscore identifier should be accepted")

-- Unicode in string literals is still fine (not identifiers)
local s = "café"
assert(#s == 5, "Unicode string should preserve bytes")

print("OK")
