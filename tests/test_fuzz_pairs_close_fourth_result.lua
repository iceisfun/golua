-- test_fuzz_pairs_close_fourth_result:
-- Lua 5.5 closes the fourth value returned by __pairs after the generic-for
-- loop finishes.

local closed = false
local closer = setmetatable({}, {
  __close = function()
    closed = true
  end,
})

local t = setmetatable({}, {
  __pairs = function()
    return function(_, control)
      if control == nil then
        return 1, "v"
      end
    end, "state", nil, closer
  end,
})

local seen = false
for k, v in pairs(t) do
  assert(k == 1 and v == "v", "unexpected iterator values")
  seen = true
end

assert(seen, "iterator should have produced one value")
assert(closed, "fourth __pairs result should be closed after loop")

print("ok")
