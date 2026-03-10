-- Fix 1: global unpack should not exist in Lua 5.4
assert(type(unpack) == "nil", "global unpack should be nil, got " .. type(unpack))
assert(type(table.unpack) == "function", "table.unpack should be a function")

-- Fix 2: xpcall message handler should receive error with file:line prefix
local ok, err = xpcall(
  function() table.insert({}, 5, "x") end,
  function(e) return "H:" .. tostring(e) end
)
assert(not ok)
assert(err:match("H:.*:%d+:"), "expected file:line prefix in xpcall handler error, got: " .. tostring(err))

print("OK")
