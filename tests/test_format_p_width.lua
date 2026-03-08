local t = {}
-- Test width
local s = string.format("%20p", t)
assert(#s == 20, "expected width 20, got " .. #s)

-- Test left-align
local s2 = string.format("%-20p", t)
assert(#s2 == 20, "expected width 20, got " .. #s2)
assert(s2:sub(-1) == " ", "expected trailing space for left-align")

-- Test without width (should still work)
local s3 = string.format("%p", t)
assert(#s3 > 0, "should produce non-empty output")

print("OK")
