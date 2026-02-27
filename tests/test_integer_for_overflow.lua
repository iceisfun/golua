-- Bug: Integer for loops that reach maxinteger/mininteger overflow
-- and loop infinitely instead of terminating correctly.

local MAX = math.maxinteger
local MIN = math.mininteger

-- Test 1: maxinteger-1 to maxinteger (should iterate exactly 2 times)
local c1 = 0
for i = MAX - 1, MAX, 1 do
  c1 = c1 + 1
  if c1 > 10 then break end  -- safety guard
end
assert(c1 == 2, "maxinteger-1 to maxinteger should iterate 2 times, got " .. c1)

-- Test 2: mininteger+1 to mininteger with step -1 (should iterate 2 times)
local c2 = 0
for i = MIN + 1, MIN, -1 do
  c2 = c2 + 1
  if c2 > 10 then break end
end
assert(c2 == 2, "mininteger+1 to mininteger step -1 should iterate 2 times, got " .. c2)

-- Test 3: maxinteger to maxinteger (should iterate exactly 1 time)
local c3 = 0
for i = MAX, MAX, 1 do
  c3 = c3 + 1
  if c3 > 10 then break end
end
assert(c3 == 1, "maxinteger to maxinteger should iterate 1 time, got " .. c3)

-- Test 4: mininteger to mininteger (should iterate exactly 1 time)
local c4 = 0
for i = MIN, MIN, -1 do
  c4 = c4 + 1
  if c4 > 10 then break end
end
assert(c4 == 1, "mininteger to mininteger should iterate 1 time, got " .. c4)

-- Test 5: step > range (single iteration)
local c5 = 0
for i = MAX - 5, MAX, 100 do
  c5 = c5 + 1
  if c5 > 10 then break end
end
assert(c5 == 1, "step > range should iterate 1 time, got " .. c5)

-- Test 6: step > range negative direction
local c6 = 0
for i = MIN + 5, MIN, -100 do
  c6 = c6 + 1
  if c6 > 10 then break end
end
assert(c6 == 1, "step > range negative should iterate 1 time, got " .. c6)

-- Test 7: 0 to maxinteger with step maxinteger (should iterate 2 times)
local c7 = 0
for i = 0, MAX, MAX do
  c7 = c7 + 1
  if c7 > 10 then break end
end
assert(c7 == 2, "0 to maxinteger step maxinteger should iterate 2 times, got " .. c7)

-- Test 8: 0 to mininteger with step mininteger (should iterate 2 times)
local c8 = 0
for i = 0, MIN, MIN do
  c8 = c8 + 1
  if c8 > 10 then break end
end
assert(c8 == 2, "0 to mininteger step mininteger should iterate 2 times, got " .. c8)

print("PASS")
