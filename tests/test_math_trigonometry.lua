-- test_math_trigonometry: sin, cos, tan, asin, acos, atan, rad, deg

-- sin/cos/tan
do
    assert(not pcall(math.sin), "sin() no args")
    assert(not pcall(math.cos), "cos() no args")
    assert(not pcall(math.tan), "tan() no args")
    assert(not pcall(math.sin, {}), "sin({}) should error")
    assert(not pcall(math.cos, {}), "cos({}) should error")
    assert(not pcall(math.tan, {}), "tan({}) should error")

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

    assert(sin(0) == 0, "sin(0)")
    assert(sin(pi/2) == 1, "sin(pi/2)")
    assert(sin(pi) == 0, "sin(pi)")
    assert(sin(3*pi/2) == -1, "sin(3pi/2)")

    assert(cos(0) == 1, "cos(0)")
    assert(cos(pi/2) == 0, "cos(pi/2)")
    assert(cos(pi) == -1, "cos(pi)")
    assert(cos(3*pi/2) == 0, "cos(3pi/2)")

    assert(tan(0) == 0, "tan(0)")
    assert(tan(pi/2) == 1/0, "tan(pi/2)")
    assert(tan(pi) == 0, "tan(pi)")
    assert(tan(3*pi/2) == 1/0, "tan(3pi/2)")
end

-- asin/acos/atan
do
    assert(not pcall(math.asin), "asin() no args")
    assert(not pcall(math.acos), "acos() no args")
    assert(not pcall(math.atan), "atan() no args")
    assert(not pcall(math.asin, {}), "asin({}) should error")
    assert(not pcall(math.acos, {}), "acos({}) should error")
    assert(not pcall(math.atan, {}), "atan({}) should error")

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

    assert(acos(-1) == 4, "acos(-1)")
    assert(acos(0) == 2, "acos(0)")
    assert(acos(1) == 0, "acos(1)")

    assert(asin(-1) == -2, "asin(-1)")
    assert(asin(0) == 0, "asin(0)")
    assert(asin(1) == 2, "asin(1)")

    assert(atan(-1) == -1, "atan(-1)")
    assert(atan(0) == 0, "atan(0)")
    assert(atan(1) == 1, "atan(1)")

    assert(math.atan(1, 0) == math.pi/2, "atan(1, 0) == pi/2")
end

-- rad/deg
do
    assert(math.rad(90) == math.pi / 2, "rad(90)")
    assert(math.deg(math.pi) == 180, "deg(pi)")
    assert(not pcall(math.rad), "rad() no args")
    assert(not pcall(math.deg), "deg() no args")
    assert(not pcall(math.rad, {}), "rad({}) should error")
    assert(not pcall(math.deg, {}), "deg({}) should error")
end
