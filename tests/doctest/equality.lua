-- Equality semantics across all Lua types
-- Tests ==, ~=, rawequal for every type combination

---------------------------------------------------------------------
-- nil equality
---------------------------------------------------------------------
print(nil == nil)
--> =true
print(nil ~= nil)
--> =false
print(nil == false)
--> =false
print(nil == 0)
--> =false
print(nil == "")
--> =false

---------------------------------------------------------------------
-- boolean equality
---------------------------------------------------------------------
print(true == true)
--> =true
print(false == false)
--> =true
print(true == false)
--> =false
print(true ~= false)
--> =true
print(false == nil)
--> =false
print(true == 1)
--> =false
print(false == 0)
--> =false

---------------------------------------------------------------------
-- integer equality
---------------------------------------------------------------------
print(1 == 1)
--> =true
print(1 == 2)
--> =false
print(1 ~= 2)
--> =true
print(0 == 0)
--> =true
print(-1 == -1)
--> =true
print(math.maxinteger == math.maxinteger)
--> =true
print(math.mininteger == math.mininteger)
--> =true

---------------------------------------------------------------------
-- float equality
---------------------------------------------------------------------
print(1.0 == 1.0)
--> =true
print(1.5 == 1.5)
--> =true
print(1.0 == 2.0)
--> =false
print(0.0 == -0.0)
--> =true
print(math.huge == math.huge)
--> =true
print(-math.huge == -math.huge)
--> =true

-- NaN is never equal to anything
print(0/0 == 0/0)
--> =false
print(0/0 ~= 0/0)
--> =true

---------------------------------------------------------------------
-- integer/float cross-type equality
---------------------------------------------------------------------
print(1 == 1.0)
--> =true
print(2 == 2.0)
--> =true
print(0 == 0.0)
--> =true
print(-1 == -1.0)
--> =true
-- Large integer that loses precision as float
print(math.maxinteger == math.maxinteger + 0.0)
--> =false

---------------------------------------------------------------------
-- string equality
---------------------------------------------------------------------
print("hello" == "hello")
--> =true
print("hello" == "world")
--> =false
print("" == "")
--> =true
print("hello" ~= "world")
--> =true
print("hello" == 42)
--> =false
print("42" == 42)
--> =false

---------------------------------------------------------------------
-- table equality (identity)
---------------------------------------------------------------------
local t1 = {}
local t2 = {}
local t3 = t1
print(t1 == t1)
--> =true
print(t1 == t3)
--> =true
print(t1 == t2)
--> =false
print(t1 ~= t2)
--> =true
print({} == {})
--> =false

---------------------------------------------------------------------
-- Lua function equality (identity)
---------------------------------------------------------------------
local function f1() end
local function f2() end
local f3 = f1
print(f1 == f1)
--> =true
print(f1 == f3)
--> =true
print(f1 == f2)
--> =false
print(f1 ~= f2)
--> =true

-- Closures from same factory are different objects
local function make() return function() end end
local c1 = make()
local c2 = make()
print(c1 == c1)
--> =true
print(c1 == c2)
--> =false

---------------------------------------------------------------------
-- Native function equality (identity)
-- This is the critical test: GoLua panics on native func ==
---------------------------------------------------------------------
print(print == print)
--> =true
print(tostring == tostring)
--> =true
print(print == tostring)
--> =false
print(print ~= tostring)
--> =true
print(type == type)
--> =true

-- Native function compared with other types
print(print == nil)
--> =false
print(print == true)
--> =false
print(print == 42)
--> =false
print(print == "print")
--> =false
print(print == {})
--> =false

-- Native function stored in table
local t = {f = print}
print(t.f == print)
--> =true
print(t.f == tostring)
--> =false

-- Native vs Lua function
local function g() end
print(g == print)
--> =false
print(print == g)
--> =false

---------------------------------------------------------------------
-- Cross-type equality (always false for incompatible types)
---------------------------------------------------------------------
print(1 == "1")
--> =false
print(true == 1)
--> =false
print(nil == false)
--> =false
print({} == "table")
--> =false
print(print == "function")
--> =false

---------------------------------------------------------------------
-- rawequal (bypasses metamethods)
---------------------------------------------------------------------
print(rawequal(nil, nil))
--> =true
print(rawequal(true, true))
--> =true
print(rawequal(false, false))
--> =true
print(rawequal(1, 1))
--> =true
print(rawequal(1, 1.0))
--> =true
print(rawequal(1.0, 1.0))
--> =true
print(rawequal("a", "a"))
--> =true
print(rawequal(print, print))
--> =true
print(rawequal(print, tostring))
--> =false

-- rawequal ignores __eq metamethod
local mt = {__eq = function() return true end}
local a = setmetatable({}, mt)
local b = setmetatable({}, mt)
print(a == b)
--> =true
print(rawequal(a, b))
--> =false

---------------------------------------------------------------------
-- __eq metamethod
---------------------------------------------------------------------
local mt2 = {__eq = function(x, y) return true end}
local x = setmetatable({}, mt2)
local y = setmetatable({}, mt2)
print(x == y)
--> =true
print(x ~= y)
--> =false

-- __eq with different metatables: uses first operand's __eq
local mt_a = {__eq = function() return true end}
local mt_b = {__eq = function() return false end}
local ea = setmetatable({}, mt_a)
local eb = setmetatable({}, mt_b)
print(ea == eb)
--> =true

-- __eq only triggers when both are tables (or same type with metamethods)
local mt3 = {__eq = function() return true end}
local v = setmetatable({}, mt3)
print(v == 42)
--> =false
print(v == "hello")
--> =false
print(v == nil)
--> =false
