-- Test that xpcall message handler runs BEFORE to-be-closed variables
-- In Lua 5.4, the order is: error -> handler -> __close
-- This was a bug where golua ran __close before the handler.

local log = {}
local ok, result = xpcall(function()
  local x <close> = setmetatable({}, {
    __close = function() log[#log+1] = "CLOSE" end
  })
  error("boom")
end, function(e) log[#log+1] = "HANDLER"; return e end)

assert(not ok)
local order = table.concat(log, " -> ")
assert(order == "HANDLER -> CLOSE", "expected 'HANDLER -> CLOSE' but got '" .. order .. "'")

-- Test 2: handler can see state before cleanup
local state = "alive"
local handler_saw = nil
xpcall(function()
  local x <close> = setmetatable({}, {
    __close = function() state = "closed" end
  })
  error("boom")
end, function(e)
  handler_saw = state
  return e
end)
assert(handler_saw == "alive", "handler should see state before __close, got: " .. tostring(handler_saw))

-- Test 3: if handler errors, TBC still closes
local tbc_closed = false
local ok3, msg3 = xpcall(function()
  local x <close> = setmetatable({}, {
    __close = function() tbc_closed = true end
  })
  error("original")
end, function(e)
  error("handler error")
end)
assert(not ok3)
assert(tbc_closed, "TBC should still close even when handler errors")

print("PASSED")
