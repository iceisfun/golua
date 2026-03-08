-- Test that callable tables (tables with __call metamethod) work as metamethod values
-- In Lua 5.4, metamethod values are invoked through the same call mechanism,
-- so a table with __call should work as any metamethod value.

-- Helper: create a callable table that returns a fixed value
local function makeCallable(retval)
    local t = {}
    setmetatable(t, { __call = function(self, ...)
        return retval
    end })
    return t
end

-- Helper: create a callable table that returns all args concatenated
local function makeCallableConcat()
    local t = {}
    setmetatable(t, { __call = function(self, a, b)
        return tostring(a) .. "+" .. tostring(b)
    end })
    return t
end

----------------------------------------------------------------------
-- Arithmetic metamethods
----------------------------------------------------------------------
do
    local mt = {
        __add = makeCallable(10),
        __sub = makeCallable(20),
        __mul = makeCallable(30),
        __div = makeCallable(40),
        __mod = makeCallable(50),
        __unm = makeCallable(60),
        __idiv = makeCallable(70),
        __pow = makeCallable(80),
    }
    local a = setmetatable({}, mt)

    assert(a + 1 == 10, "__add with callable table failed")
    assert(a - 1 == 20, "__sub with callable table failed")
    assert(a * 1 == 30, "__mul with callable table failed")
    assert(a / 1 == 40, "__div with callable table failed")
    assert(a % 1 == 50, "__mod with callable table failed")
    assert(-a == 60, "__unm with callable table failed")
    assert(a // 1 == 70, "__idiv with callable table failed")
    assert(a ^ 1 == 80, "__pow with callable table failed")
    print("PASS: arithmetic metamethods with callable tables")
end

----------------------------------------------------------------------
-- Bitwise metamethods
----------------------------------------------------------------------
do
    local mt = {
        __band = makeCallable(100),
        __bor = makeCallable(200),
        __bxor = makeCallable(300),
        __bnot = makeCallable(400),
        __shl = makeCallable(500),
        __shr = makeCallable(600),
    }
    local a = setmetatable({}, mt)

    assert(a & 1 == 100, "__band with callable table failed")
    assert(a | 1 == 200, "__bor with callable table failed")
    assert(a ~ 1 == 300, "__bxor with callable table failed")
    assert(~a == 400, "__bnot with callable table failed")
    assert(a << 1 == 500, "__shl with callable table failed")
    assert(a >> 1 == 600, "__shr with callable table failed")
    print("PASS: bitwise metamethods with callable tables")
end

----------------------------------------------------------------------
-- Comparison metamethods
----------------------------------------------------------------------
do
    local eqCalled = false
    local ltCalled = false
    local leCalled = false

    local eqMM = setmetatable({}, { __call = function(self, a, b)
        eqCalled = true
        return true
    end })
    local ltMM = setmetatable({}, { __call = function(self, a, b)
        ltCalled = true
        return true
    end })
    local leMM = setmetatable({}, { __call = function(self, a, b)
        leCalled = true
        return true
    end })

    local mt = { __eq = eqMM, __lt = ltMM, __le = leMM }
    local a = setmetatable({}, mt)
    local b = setmetatable({}, mt)

    assert(a == b, "__eq with callable table failed")
    assert(eqCalled, "__eq callable table was not called")

    assert(a < b, "__lt with callable table failed")
    assert(ltCalled, "__lt callable table was not called")

    assert(a <= b, "__le with callable table failed")
    assert(leCalled, "__le callable table was not called")
    print("PASS: comparison metamethods with callable tables")
end

----------------------------------------------------------------------
-- __concat metamethod
----------------------------------------------------------------------
do
    local mt = { __concat = makeCallableConcat() }
    local a = setmetatable({}, mt)
    -- The callable table's __call returns tostring(a).."+".."hello"
    local result = a .. "hello"
    assert(type(result) == "string", "__concat with callable table failed, got " .. type(result))
    print("PASS: __concat with callable table")
end

----------------------------------------------------------------------
-- __len metamethod
----------------------------------------------------------------------
do
    local mt = { __len = makeCallable(42) }
    local a = setmetatable({}, mt)
    assert(#a == 42, "__len with callable table failed, got " .. tostring(#a))
    print("PASS: __len with callable table")
end

----------------------------------------------------------------------
-- __close metamethod
----------------------------------------------------------------------
do
    local closeCalled = false
    local closeMM = setmetatable({}, { __call = function(self, obj, err)
        closeCalled = true
    end })

    do
        local a <close> = setmetatable({}, { __close = closeMM })
    end
    assert(closeCalled, "__close with callable table was not called")
    print("PASS: __close with callable table")
end

----------------------------------------------------------------------
-- Deeply nested callable tables (__call itself is a callable table)
----------------------------------------------------------------------
do
    -- inner callable table
    local inner = setmetatable({}, { __call = function(self, outerSelf, a, b)
        return 999
    end })
    -- outer callable table whose __call is the inner callable table
    local outer = setmetatable({}, { __call = inner })

    local mt = { __add = outer }
    local a = setmetatable({}, mt)
    assert(a + 1 == 999, "deeply nested callable table as __add failed")
    print("PASS: deeply nested callable tables")
end

print("ALL TESTS PASSED")
