-- Bug: Lua 5.4 generic for loop supports a 4th return value from
-- the iterator factory which serves as a to-be-closed variable.
-- golua ignores this 4th value entirely.

local closed = false

local function closeable_iter(t)
  local i = 0
  local obj = setmetatable({}, {
    __close = function()
      closed = true
    end
  })
  return function()
    i = i + 1
    if i <= #t then return i, t[i] end
  end, nil, nil, obj  -- 4th value is to-be-closed
end

local results = {}
for i, v in closeable_iter({10, 20, 30}) do
  results[#results + 1] = v
end

assert(#results == 3, "should iterate 3 times")
assert(results[1] == 10 and results[2] == 20 and results[3] == 30)
assert(closed, "4th return value should be closed when loop ends")

print("PASS")
