-- Numeric for loop should validate operands in Lua 5.4 order: limit, step, initial
-- Bug: GoLua validated initial first

-- All three bad: should report limit
local ok, err = pcall(function() for i = "a", "b", "c" do end end)
assert(not ok)
assert(string.find(err, "bad 'for' limit"), "expected limit error, got: " .. tostring(err))

-- Initial bad, limit ok, step bad: should report step
ok, err = pcall(function() for i = "a", 10, "c" do end end)
assert(not ok)
assert(string.find(err, "bad 'for' step"), "expected step error, got: " .. tostring(err))

-- Initial ok, limit bad, step bad: should report limit
ok, err = pcall(function() for i = 1, "b", "c" do end end)
assert(not ok)
assert(string.find(err, "bad 'for' limit"), "expected limit error, got: " .. tostring(err))

-- Only initial bad: should report initial
ok, err = pcall(function() for i = "a", 10, 1 do end end)
assert(not ok)
assert(string.find(err, "bad 'for' initial"), "expected initial error, got: " .. tostring(err))

print("PASS")
