-- Test: integer for-loop counter overflow
-- When the unsigned iteration count exceeds MaxInt64, loops should still work

-- Loop from -2 to maxinteger with step 1: huge iteration count
local c = 0
for i = -2, math.maxinteger do
    c = c + 1
    if c > 3 then break end
end
assert(c == 4, "expected 4 iterations, got " .. c)

-- Loop from mininteger to maxinteger with step 1: maximum possible iteration count
c = 0
for i = math.mininteger, math.maxinteger do
    c = c + 1
    if c > 3 then break end
end
assert(c == 4, "expected 4 iterations, got " .. c)

-- Simple loop should still work
c = 0
for i = 1, 5 do c = c + 1 end
assert(c == 5, "expected 5 iterations, got " .. c)

-- Negative step
c = 0
for i = 5, 1, -1 do c = c + 1 end
assert(c == 5, "expected 5 iterations, got " .. c)

print("OK")
