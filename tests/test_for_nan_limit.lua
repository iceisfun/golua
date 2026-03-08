-- Test: NaN as for-loop limit runs 0 iterations (not an error)

-- NaN limit with integer init/step
local c = 0
for i = 1, 0/0 do c = c + 1 end
assert(c == 0, "NaN limit should run 0 iterations, got " .. c)

-- NaN limit with float init/step
c = 0
for i = 1.0, 0/0 do c = c + 1 end
assert(c == 0, "NaN float limit should run 0 iterations")

-- NaN limit with negative step
c = 0
for i = 10, 0/0, -1 do c = c + 1 end
assert(c == 0, "NaN limit with negative step should run 0 iterations")

-- Non-NaN cases still work
c = 0
for i = 1, 5 do c = c + 1 end
assert(c == 5, "normal loop should run 5 iterations")

print("OK")
