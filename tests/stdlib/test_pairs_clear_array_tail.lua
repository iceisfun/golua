-- Test: clearing the last array key during a traversal
-- What: clearing t[#t] while pairs()/next() is parked on it shrinks golua's
-- array part. A stale hash tombstone for that index (a hash key deleted before
-- the array grew over it, or a slot left by promotion) used to make next()
-- resume after the tombstone and skip the live hash keys in front of it.

local function count(t)
  local n = 0
  for _ in pairs(t) do n = n + 1 end
  return n
end

-- A hash key 3 deleted before the array grew over index 3.
do
  local t = {}
  t.a, t.b, t.c = 1, 2, 3
  t[3] = true
  t[3] = nil
  t[1] = 1 t[2] = 2 t[3] = 3 -- in this order: the array grows over 3
  local seen = {}
  local k = next(t)
  while k ~= nil do
    seen[#seen + 1] = tostring(k)
    if k == 3 then t[3] = nil end
    k = next(t, k)
  end
  assert(#seen == 6, "visited " .. table.concat(seen, " "))
end

-- Evens then odds leaves promotion tombstones between two string keys; the
-- clear-everything idiom must still reach the first string key.
do
  local t = {}
  t.x = 1
  for i = 2, 10, 2 do t[i] = i end
  t.y = 2
  for i = 1, 10, 2 do t[i] = i end
  assert(count(t) == 12)
  local n = 0
  for k in pairs(t) do t[k] = nil n = n + 1 end
  assert(n == 12, "cleared " .. n .. " keys")
  assert(next(t) == nil)
end

-- The same past the size at which the tombstones are found through an index.
do
  local t = {}
  for i = 1, 40 do t["s" .. i] = i end
  for i = 2, 64, 2 do t[i] = i end
  t.y = 2
  for i = 1, 64, 2 do t[i] = i end
  local n = 0
  for k in pairs(t) do t[k] = nil n = n + 1 end
  assert(n == 105, "cleared " .. n .. " keys")
  assert(next(t) == nil)
end

-- A hash key cleared while the traversal is parked on it, beyond the array
-- part but inside the array's spare capacity, must still resume in place.
do
  local t = {1}
  t[3] = 3
  t.z = 1
  local n = 0
  for k in pairs(t) do t[k] = nil n = n + 1 end
  assert(n == 3, "cleared " .. n .. " keys")
  assert(next(t) == nil)
end
