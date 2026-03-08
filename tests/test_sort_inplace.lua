-- Test: table.sort must sort in-place through metamethods, not copy-sort-writeback
-- This verifies reads > N and writes > N when sorting through a proxy table,
-- which proves the sort algorithm reads/writes individual elements during sorting
-- rather than bulk-copying to a Go slice.

-- Test 1: Proxy table metamethod read/write counts
do
  local reads, writes = 0, 0
  local data = {5, 3, 1, 4, 2}
  local proxy = setmetatable({}, {
    __index = function(t, k) reads = reads + 1; return data[k] end,
    __newindex = function(t, k, v) writes = writes + 1; data[k] = v end,
    __len = function() return #data end
  })
  table.sort(proxy)
  -- In-place sort must read more than N elements (comparisons read from table)
  assert(reads > 5, "reads should be > 5 for in-place sort, got " .. reads)
  -- In-place sort must write more than N elements (swaps write to table)
  assert(writes > 5, "writes should be > 5 for in-place sort, got " .. writes)
  -- Result must still be sorted
  for i = 1, 4 do
    assert(data[i] <= data[i+1], "data not sorted at index " .. i)
  end
  print("PASS: proxy table read/write counts")
end

-- Test 2: Table mutation during sort comparator persists
do
  local t = {3, 1, 4, 1, 5}
  local count = 0
  table.sort(t, function(a, b)
    count = count + 1
    if count == 2 then
      t[5] = 99  -- mutate during sort
    end
    return a < b
  end)
  -- In Lua 5.4, the mutation persists because the sort operates on the live table
  -- With copy-sort-writeback, the writeback overwrites the mutation
  assert(t[5] == 99, "mutation during sort should persist, got t[5]=" .. tostring(t[5]))
  print("PASS: mutation during sort persists")
end

-- Test 3: Basic sorting still works correctly
do
  local t = {5, 3, 1, 4, 2}
  table.sort(t)
  for i = 1, 4 do
    assert(t[i] <= t[i+1], "not sorted at " .. i)
  end
  assert(t[1] == 1 and t[5] == 5)
  print("PASS: basic sorting")
end

-- Test 4: Sort with custom comparator (reverse)
do
  local t = {1, 2, 3, 4, 5}
  table.sort(t, function(a, b) return a > b end)
  for i = 1, 4 do
    assert(t[i] >= t[i+1], "not reverse sorted at " .. i)
  end
  assert(t[1] == 5 and t[5] == 1)
  print("PASS: custom comparator")
end

-- Test 5: Sort with __lt metamethod
do
  local mt = {__lt = function(a, b) return a.val < b.val end}
  local t = {}
  for i = 1, 5 do t[i] = setmetatable({val = 6 - i}, mt) end
  table.sort(t)
  for i = 1, 4 do
    assert(t[i].val <= t[i+1].val, "not sorted by __lt at " .. i)
  end
  print("PASS: __lt metamethod")
end

-- Test 6: Error propagation from comparator
do
  local t = {3, 1, 2}
  local ok, err = pcall(table.sort, t, function(a, b)
    error("comp error")
  end)
  assert(not ok, "should error")
  assert(tostring(err):find("comp error"), "error message should contain 'comp error'")
  print("PASS: error propagation")
end

-- Test 7: Proxy table with custom comparator
do
  local reads = 0
  local data = {5, 3, 1, 4, 2}
  local proxy = setmetatable({}, {
    __index = function(t, k) reads = reads + 1; return data[k] end,
    __newindex = function(t, k, v) data[k] = v end,
    __len = function() return #data end
  })
  table.sort(proxy, function(a, b) return a > b end)
  -- Should still have many reads even with custom comparator
  assert(reads > 5, "reads with custom comp should be > 5, got " .. reads)
  for i = 1, 4 do
    assert(data[i] >= data[i+1], "not reverse sorted at " .. i)
  end
  print("PASS: proxy with custom comparator")
end

print("All sort in-place tests passed")
