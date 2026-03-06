-- Comprehensive string.dump round-trip tests

-- Simple function
local function square(x) return x * x end
local d = string.dump(square)
local f = load(d)
assert(f(5) == 25)
assert(f(0) == 0)
assert(f(-3) == 9)

-- Stripped dump
local ds = string.dump(square, true)
local fs = load(ds)
assert(fs(5) == 25)
assert(#ds <= #d, "stripped should be <= unstripped")

-- Multiple return values
local function multi() return 1, "two", true, nil, 3.14 end
local dm = string.dump(multi)
local fm = load(dm)
local a, b, c, d2, e = fm()
assert(a == 1 and b == "two" and c == true and d2 == nil and e == 3.14)

-- All constant types
local function allconst()
    return nil, true, false, 42, -100, 3.14159, 0.0, "hello", ""
end
local da = string.dump(allconst)
local fa = load(da)
local r1, r2, r3, r4, r5, r6, r7, r8, r9 = fa()
assert(r1 == nil)
assert(r2 == true)
assert(r3 == false)
assert(r4 == 42)
assert(r5 == -100)
assert(r6 == 3.14159)
assert(r7 == 0.0)
assert(r8 == "hello")
assert(r9 == "")

-- Vararg function
local function va(...)
    return select("#", ...), ...
end
local dv = string.dump(va)
local fv = load(dv)
local n, x, y = fv(10, 20)
assert(n == 2 and x == 10 and y == 20)
n = fv()
assert(n == 0)

-- Nested function (closure creation)
local function make_counter(start)
    local n = start
    return function()
        n = n + 1
        return n
    end
end
local dc = string.dump(make_counter)
local fc = load(dc)
local counter = fc(10)
assert(counter() == 11)
assert(counter() == 12)
assert(counter() == 13)

-- Loops: for, while, repeat
local function loops()
    local s = 0
    for i = 1, 10 do s = s + i end
    local t = 0
    local i = 1
    while i <= 5 do t = t + i; i = i + 1 end
    local u = 0
    local j = 1
    repeat u = u + j; j = j + 1 until j > 3
    return s, t, u
end
local dl = string.dump(loops)
local fl = load(dl)
local s, t, u = fl()
assert(s == 55 and t == 15 and u == 6)

-- Table constructor
local function tables()
    local t = {1, 2, 3, x = "hello", [true] = false}
    return #t, t.x, t[true]
end
local dt = string.dump(tables)
local ft = load(dt)
local len, x, b = ft()
assert(len == 3 and x == "hello" and b == false)

-- If/elseif/else
local function classify(n)
    if n > 0 then return "pos"
    elseif n < 0 then return "neg"
    else return "zero" end
end
local dcl = string.dump(classify)
local fcl = load(dcl)
assert(fcl(1) == "pos")
assert(fcl(-1) == "neg")
assert(fcl(0) == "zero")

-- String operations (uses upvalue _ENV.string)
local function strops(s)
    return string.upper(s) .. "!" .. #s
end
local dso = string.dump(strops)
local fso = load(dso)
assert(fso("hello") == "HELLO!5")

-- Large function with many constants
local function large()
    local a = 1; local b = 2; local c = 3; local d2 = 4; local e = 5
    local f = 6; local g = 7; local h = 8; local i = 9; local j = 10
    return a+b+c+d2+e+f+g+h+i+j
end
local dlg = string.dump(large)
local flg = load(dlg)
assert(flg() == 55)

-- Empty function
local function noop() end
local dn = string.dump(noop)
local fn2 = load(dn)
fn2() -- should not error

-- Function with many parameters
local function many(a,b,c,d2,e,f,g,h)
    return a+b+c+d2+e+f+g+h
end
local dmn = string.dump(many)
local fmn = load(dmn)
assert(fmn(1,2,3,4,5,6,7,8) == 36)

-- Extreme integer constants
local function extremes()
    return 9223372036854775807, -9223372036854775808, 0
end
local de = string.dump(extremes)
local fe = load(de)
local mx, mn, z = fe()
assert(mx == math.maxinteger)
assert(mn == math.mininteger)
assert(z == 0)

-- Extreme float constants
local function floats()
    return 1/0, -1/0, 0.0, -0.0, 1e308
end
local df = string.dump(floats)
local ff = load(df)
local inf, ninf, zero, nzero, big = ff()
assert(inf == math.huge)
assert(ninf == -math.huge)
assert(zero == 0.0)
assert(big == 1e308)

-- Environment override
local env = setmetatable({myvar = 999}, {__index = _G})
local function use_env() return myvar end
local due = string.dump(use_env)
local fue = load(due, nil, nil, env)
assert(fue() == 999, "env override should work")

-- Chunkname for binary chunk
local dck = string.dump(function() return 1 end)
local fck = load(dck, "=mychunk")
assert(fck ~= nil and fck() == 1)

-- Mode checks
local dmode = string.dump(function() return 42 end)
-- "b" should accept binary
local fb2 = load(dmode, nil, "b")
assert(fb2 ~= nil and fb2() == 42)
-- "t" should reject binary
local ft2, et = load(dmode, nil, "t")
assert(ft2 == nil and et:find("binary"))
-- "bt" should accept binary
local fbt = load(dmode, nil, "bt")
assert(fbt ~= nil and fbt() == 42)

-- Dump function with upvalues (should succeed)
local upval = 42
local function with_up() return upval end
local dup = string.dump(with_up)
assert(type(dup) == "string" and #dup > 0)

print("PASS")
