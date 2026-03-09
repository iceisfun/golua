-- Setting t[len+1] when that key already exists in integer hash must not
-- leave a duplicate key path for pairs/next iteration.

do
  local t = {8, 5, 9, 4, ["c"] = false, [5] = "x", [2] = 8}

  local seen = 0
  for k, v in pairs(t) do
    seen = seen + 1
    if type(k) == "number" then
      t[k] = v
    end
    assert(seen <= 20, "pairs loop should terminate, saw " .. tostring(seen))
  end

  -- Stable finite traversal: 1..5 and "c".
  assert(seen == 6, "expected 6 entries, got " .. tostring(seen))

  -- next() traversal should also terminate without repeating key 5.
  local k = nil
  local n = 0
  local seen5 = 0
  while true do
    k = next(t, k)
    if k == nil then
      break
    end
    n = n + 1
    if k == 5 then
      seen5 = seen5 + 1
    end
    assert(n <= 20, "next loop should terminate, saw " .. tostring(n))
  end
  assert(n == 6, "expected 6 next() entries, got " .. tostring(n))
  assert(seen5 == 1, "key 5 should appear once, got " .. tostring(seen5))
end
