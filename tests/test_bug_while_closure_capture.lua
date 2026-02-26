-- Bug: closures in while/repeat loops have corrupted upvalues after loop exit.
-- Inside the loop, captured locals work. After the loop, they return the
-- closure itself instead of the captured value.

-- While loop: capture local per iteration
local fns = {}
local i = 0
while i < 3 do
    i = i + 1
    local cap = i
    fns[i] = function() return cap end
end
assert(fns[1]() == 1, "while capture iter 1: got " .. tostring(fns[1]()))
assert(fns[2]() == 2, "while capture iter 2: got " .. tostring(fns[2]()))
assert(fns[3]() == 3, "while capture iter 3: got " .. tostring(fns[3]()))

-- Repeat-until loop: same bug
local rfns = {}
local ri = 0
repeat
    ri = ri + 1
    local cap = ri
    rfns[ri] = function() return cap end
until ri >= 3
assert(rfns[1]() == 1, "repeat capture iter 1: got " .. tostring(rfns[1]()))
assert(rfns[2]() == 2, "repeat capture iter 2: got " .. tostring(rfns[2]()))
assert(rfns[3]() == 3, "repeat capture iter 3: got " .. tostring(rfns[3]()))

-- While with explicit do-end block inside
local dfns = {}
local di = 0
while di < 3 do
    di = di + 1
    do
        local cap = di
        dfns[di] = function() return cap end
    end
end
assert(dfns[1]() == 1, "while+do capture iter 1: got " .. tostring(dfns[1]()))
assert(dfns[2]() == 2, "while+do capture iter 2: got " .. tostring(dfns[2]()))
assert(dfns[3]() == 3, "while+do capture iter 3: got " .. tostring(dfns[3]()))

-- For-loop works (guard: should keep working)
local ffns = {}
for j = 1, 3 do
    ffns[j] = function() return j end
end
assert(ffns[1]() == 1, "for capture iter 1")
assert(ffns[2]() == 2, "for capture iter 2")
assert(ffns[3]() == 3, "for capture iter 3")

-- Single iteration: after loop the upvalue should hold the final value
local single_fn
local si = 0
while si < 1 do
    si = si + 1
    local cap = si * 10
    single_fn = function() return cap end
end
assert(single_fn() == 10, "single iter after loop: got " .. tostring(single_fn()))

-- Nested while loops: inner closures capture per-iteration locals
local outer_fns = {}
local oi = 0
while oi < 2 do
    oi = oi + 1
    local cap_oi = oi  -- per-iteration copy
    local inner_fns = {}
    local ij = 0
    while ij < 2 do
        ij = ij + 1
        local cap_o, cap_i = cap_oi, ij
        inner_fns[ij] = function() return cap_o, cap_i end
    end
    local o1, i1 = inner_fns[1]()
    local o2, i2 = inner_fns[2]()
    assert(o1 == cap_oi and i1 == 1, "nested inner[1] at outer " .. cap_oi)
    assert(o2 == cap_oi and i2 == 2, "nested inner[2] at outer " .. cap_oi)
    outer_fns[oi] = function() return cap_oi end
end
assert(outer_fns[1]() == 1, "nested outer[1]: got " .. tostring(outer_fns[1]()))
assert(outer_fns[2]() == 2, "nested outer[2]: got " .. tostring(outer_fns[2]()))
