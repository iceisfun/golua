-- test_math_atrig: asin, acos, atan

-- no args should error
assert(not pcall(math.asin), "asin() no args")
assert(not pcall(math.acos), "acos() no args")
assert(not pcall(math.atan), "atan() no args")

-- table args should error
assert(not pcall(math.asin, {}), "asin({}) should error")
assert(not pcall(math.acos, {}), "acos({}) should error")
assert(not pcall(math.atan, {}), "atan({}) should error")

-- helper: normalize to multiples of pi/4
local function round(num)
    if num >= 0 then return math.floor(num+.5)
    else return math.ceil(num-.5) end
end

local function top(f)
    return function(x)
        local y = 4*f(x)/math.pi
        local ry = round(y)
        if math.abs(y - ry) > 1e-15 then
            return ry
        end
        return y
    end
end

local acos = top(math.acos)
local asin = top(math.asin)
local atan = top(math.atan)

-- acos
assert(acos(-1) == 4, "acos(-1)")
assert(acos(0) == 2, "acos(0)")
assert(acos(1) == 0, "acos(1)")

-- asin
assert(asin(-1) == -2, "asin(-1)")
assert(asin(0) == 0, "asin(0)")
assert(asin(1) == 2, "asin(1)")

-- atan
assert(atan(-1) == -1, "atan(-1)")
assert(atan(0) == 0, "atan(0)")
assert(atan(1) == 1, "atan(1)")

-- atan with two args
assert(math.atan(1, 0) == math.pi/2, "atan(1, 0) == pi/2")
