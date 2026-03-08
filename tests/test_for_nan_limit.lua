-- Test: NaN in for-loop operands, matching Lua 5.4 semantics.
-- Integer mode with positive step + NaN limit → 0 iterations.
-- Float mode uses C-style NaN comparison (NaN comparisons return false),
-- so the "should we skip?" check fails and the loop enters.

-- NaN limit with integer init/step, positive step → 0 iterations
local c = 0
for i = 1, 0/0 do c = c + 1 end
assert(c == 0, "NaN limit (int, pos step) should run 0 iterations, got " .. c)

-- NaN limit with float init/step → enters loop (C NaN comparison)
c = 0
for i = 1.0, 0/0, 1.0 do c = c + 1; break end
assert(c == 1, "NaN float limit should enter loop, got " .. c)

-- NaN init with float → enters loop
c = 0
for i = 0/0, 10.0, 1.0 do c = c + 1; break end
assert(c == 1, "NaN float init should enter loop, got " .. c)

-- NaN limit with negative integer step → enters loop (limit = MinInt64)
c = 0
for i = 10, 0/0, -1 do c = c + 1; if c > 3 then break end end
assert(c > 0, "NaN limit with neg int step should enter loop, got " .. c)

-- Non-NaN cases still work
c = 0
for i = 1, 5 do c = c + 1 end
assert(c == 5, "normal loop should run 5 iterations")

print("OK")
