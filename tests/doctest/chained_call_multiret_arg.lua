-- Chained calls (method or function) where the last argument is a multi-ret
-- expression (function call or vararg). The compiler must reset freeReg after
-- compiling the receiver expression, otherwise compileExprMultiRet starts at
-- an inflated register and leaves a gap of stale values.

-- method chain: a:m("x"):m(f())
local a = {}
function a:m(arg)
    print(type(arg) .. ":" .. tostring(arg))
    return self
end

local function f() return 42 end

a:m("x")
--> =string:x

-- the chained call must pass f()'s return value, not a stale register
a:m("first"):m(f())
--> =string:first
--> =number:42

-- function chain: outer(x)(g())
local function outer(x)
    return function(y) return "got:" .. y end
end
local function g() return 55 end
print(outer("hi")(g()))
--> =got:55

-- method chain with vararg
local function test_vararg(...)
    a:m("v"):m(...)
end
test_vararg(77)
--> =string:v
--> =number:77
