-- Table constructor: multi-return calls/varargs should be truncated to 1 value
-- when they are NOT the last field in the constructor.

local function mv() return 10, 20, 30 end

-- Call followed by named field: truncate to 1
local t1 = {mv(); x=1}
assert(t1[1] == 10, "t1[1] should be 10, got " .. tostring(t1[1]))
assert(t1[2] == nil, "t1[2] should be nil, got " .. tostring(t1[2]))
assert(t1[3] == nil, "t1[3] should be nil, got " .. tostring(t1[3]))
assert(t1.x == 1, "t1.x should be 1")

-- Call followed by positional field: truncate to 1
local t2 = {mv(), 99}
assert(t2[1] == 10, "t2[1] should be 10, got " .. tostring(t2[1]))
assert(t2[2] == 99, "t2[2] should be 99, got " .. tostring(t2[2]))
assert(t2[3] == nil, "t2[3] should be nil, got " .. tostring(t2[3]))

-- Call as LAST field: should expand all values
local t3 = {mv()}
assert(t3[1] == 10, "t3[1] should be 10")
assert(t3[2] == 20, "t3[2] should be 20")
assert(t3[3] == 30, "t3[3] should be 30")

-- Varargs truncation
local function test_varargs(...)
    local t4 = {...; x=1}
    assert(t4[1] == 10, "t4[1] should be 10, got " .. tostring(t4[1]))
    assert(t4[2] == nil, "t4[2] should be nil, got " .. tostring(t4[2]))
    assert(t4.x == 1, "t4.x should be 1")

    local t5 = {..., 99}
    assert(t5[1] == 10, "t5[1] should be 10, got " .. tostring(t5[1]))
    assert(t5[2] == 99, "t5[2] should be 99, got " .. tostring(t5[2]))
    assert(t5[3] == nil, "t5[3] should be nil, got " .. tostring(t5[3]))
end
test_varargs(10, 20, 30)

print("PASS")
