-- Bug: Generic for-in loop does not respect __call metamethod on
-- the iterator. Direct calls work, but for-in fails with
-- "attempt to call a table value".

-- Test 1: table with __call as for-in iterator
local iter = setmetatable({n = 3}, {
  __call = function(self, s, var)
    var = (var or 0) + 1
    if var > self.n then return nil end
    return var, var * var
  end
})

-- Direct call should work (baseline)
local v1, s1 = iter(nil, 0)
assert(v1 == 1 and s1 == 1, "direct call should work")

-- For-in loop should also work
local results = {}
for i, sq in iter do
  results[#results + 1] = i .. "=" .. sq
end
assert(#results == 3, "for-in should iterate 3 times, got " .. #results)
assert(results[1] == "1=1", "first: " .. tostring(results[1]))
assert(results[2] == "2=4", "second: " .. tostring(results[2]))
assert(results[3] == "3=9", "third: " .. tostring(results[3]))

