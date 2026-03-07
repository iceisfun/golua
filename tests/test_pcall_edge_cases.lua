-- Bug 1: pcall() with zero arguments causes Go runtime panic
-- Expected: error "bad argument #1 to 'pcall' (value expected)"
local ok, err = pcall(pcall)
assert(not ok, "pcall() with no args should error")
assert(type(err) == "string", "pcall error should be a string, got " .. type(err))
assert(tostring(err):find("value expected"),
  "pcall() error should say 'value expected', got: " .. tostring(err))

-- Bug 2: assert() converts error value to string instead of preserving Lua value
-- Expected: error value should remain a number, not be converted to string
local ok2, err2 = pcall(assert, nil, 42)
assert(not ok2, "assert(nil, 42) should error")
assert(type(err2) == "number", "assert error value should be number, got " .. type(err2))
assert(err2 == 42, "assert error value should be 42")

-- assert with table error value should preserve the table
local ok3, err3 = pcall(assert, nil, {msg = "boom"})
assert(not ok3, "assert(nil, {}) should error")
assert(type(err3) == "table", "assert error should be table, got " .. type(err3))
assert(err3.msg == "boom", "assert table error should preserve fields")

-- Bug 3: assert() with no arguments gives wrong error message
local ok4, err4 = pcall(assert)
assert(not ok4, "assert() with no args should error")
assert(tostring(err4):find("value expected"),
  "assert() error should say 'value expected', got: " .. tostring(err4))

-- Bug 4: xpcall doesn't validate handler is a function
local ok5, err5 = pcall(xpcall, function() error("boom") end, nil)
assert(not ok5, "xpcall with nil handler should error immediately")
assert(tostring(err5):find("function expected"),
  "xpcall nil handler error should say 'function expected', got: " .. tostring(err5))

local ok6, err6 = pcall(xpcall, function() error("boom") end, "not a func")
assert(not ok6, "xpcall with string handler should error immediately")
assert(tostring(err6):find("function expected"),
  "xpcall string handler error should say 'function expected', got: " .. tostring(err6))

