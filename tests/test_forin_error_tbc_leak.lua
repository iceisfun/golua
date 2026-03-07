-- Regression test: for-in error via pcall must not leak TBC markers.
-- When OP_TFORCALL returns an error (non-callable iterator), the TBC
-- variable at R[A+3] must be cleaned up. Otherwise subsequent function
-- calls that reuse those stack slots crash with "metamethod 'close'".

-- Trigger for-in error inside pcall
pcall(function() for k in 42 do end end)

-- Call a function that uses enough locals to overlap the leaked TBC slot
local function f()
  local x = 1
  local y = 2
  local z = 3
  local w = 4
end
f()

-- Also test with the P helper pattern that originally exposed the bug
local function P(name, ...)
  local args = table.pack(...)
  local parts = {name .. ":"}
  for i = 1, args.n do
    parts[#parts+1] = tostring(args[i])
  end
  return table.concat(parts, " ")
end

local ok, err = pcall(function() for k in 42 do end end)
assert(not ok)
assert(P("test", err:match("for iterator")) == "test: for iterator")

