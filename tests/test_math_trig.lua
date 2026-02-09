-- test_math_trig: sin, cos, tan

-- no args should error
assert(not pcall(math.sin), "sin() no args")
assert(not pcall(math.cos), "cos() no args")
assert(not pcall(math.tan), "tan() no args")

-- table args should error
assert(not pcall(math.sin, {}), "sin({}) should error")
assert(not pcall(math.cos, {}), "cos({}) should error")
assert(not pcall(math.tan, {}), "tan({}) should error")

-- helper: snap near-zero values to zero, large values to inf
local function toz(f)
    return function(x)
        local y = f(x)
        if math.abs(y) < 1e-15 then
            return 0
        elseif math.abs(y) > 1e15 then
            return 1/0
        end
        return y
    end
end

local pi = math.pi
local sin = toz(math.sin)
local cos = toz(math.cos)
local tan = toz(math.tan)

-- sin
assert(sin(0) == 0, "sin(0)")
assert(sin(pi/2) == 1, "sin(pi/2)")
assert(sin(pi) == 0, "sin(pi)")
assert(sin(3*pi/2) == -1, "sin(3pi/2)")

-- cos
assert(cos(0) == 1, "cos(0)")
assert(cos(pi/2) == 0, "cos(pi/2)")
assert(cos(pi) == -1, "cos(pi)")
assert(cos(3*pi/2) == 0, "cos(3pi/2)")

-- tan
assert(tan(0) == 0, "tan(0)")
assert(tan(pi/2) == 1/0, "tan(pi/2)")
assert(tan(pi) == 0, "tan(pi)")
assert(tan(3*pi/2) == 1/0, "tan(3pi/2)")
