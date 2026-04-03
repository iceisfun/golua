-- Named vararg parameter tests (Lua 5.5 feature)
-- function f(... name) creates a table from varargs with n field

---------------------------------------------------------------------
-- Basic: all varargs go into the named table
---------------------------------------------------------------------
local function f1(... args)
  print(type(args), args.n, args[1], args[2], args[3])
end
f1(10, 20, 30)
--> =table	3	10	20	30

---------------------------------------------------------------------
-- With regular parameters
---------------------------------------------------------------------
local function f2(a, ... rest)
  print(a, type(rest), rest.n, rest[1], rest[2])
end
f2(1, 2, 3)
--> =1	table	2	2	3

---------------------------------------------------------------------
-- Empty varargs: n should be 0
---------------------------------------------------------------------
local function f3(... args)
  print(args.n)
end
f3()
--> =0

---------------------------------------------------------------------
-- nil arguments: n should count them
---------------------------------------------------------------------
local function f4(... args)
  print(args.n, args[1], args[2], args[3])
end
f4(nil, 42, nil)
--> =3	nil	42	nil

---------------------------------------------------------------------
-- Missing regular params get nil, vararg table gets n=0
---------------------------------------------------------------------
local function f5(a, b, ... rest)
  print(a, b, rest.n)
end
f5(1)
--> =1	nil	0

---------------------------------------------------------------------
-- ... still works alongside named vararg
---------------------------------------------------------------------
local function f6(... args)
  print(...)
  print(args[1])
end
f6(1, 2, 3)
--> =1	2	3
--> =1

---------------------------------------------------------------------
-- Method syntax with named vararg
---------------------------------------------------------------------
local t = {}
function t:method(... args)
  print(type(args), args.n)
end
t:method(1, 2)
--> =table	2

---------------------------------------------------------------------
-- Anonymous function with named vararg
---------------------------------------------------------------------
local f7 = function(... args)
  return args.n, args[1]
end
print(f7(10, 20))
--> =2	10

---------------------------------------------------------------------
-- Named vararg in nested function (as upvalue)
---------------------------------------------------------------------
local function foo(... tab1)
  return function(... tab2)
    return tab1, tab2
  end
end
local inner = foo(10, 20, 30)
local t1, t2 = inner("a", "b")
print(t1.n, t1[1], t2.n, t2[1])
--> =3	10	2	a

---------------------------------------------------------------------
-- Named vararg is read-only (const)
---------------------------------------------------------------------
local ok, err = load("return function(... t) t = 10 end")
print(ok == nil)
print(err:find("const variable 't'") ~= nil)
--> =true
--> =true

---------------------------------------------------------------------
-- Named vararg captured as upvalue is also const
---------------------------------------------------------------------
local ok2, err2 = load([[
  local function foo(... extra)
    return function(...) extra = nil end
  end
]])
print(ok2 == nil)
print(err2:find("const variable 'extra'") ~= nil)
--> =true
--> =true

---------------------------------------------------------------------
-- With two regular params and named vararg
---------------------------------------------------------------------
local function f8(a, b, ... rest)
  print(a, b, rest.n, rest[1])
end
f8(1, 2, 3, 4)
--> =1	2	2	3

---------------------------------------------------------------------
-- Modifying named vararg table is visible via ...
---------------------------------------------------------------------
local function f9(... args)
  args[2] = 99
  print(select(2, ...))
end
f9(1, 2, 3)
--> =99	3

---------------------------------------------------------------------
-- Modifying n changes select("#", ...)
---------------------------------------------------------------------
local function f10(... args)
  print(select("#", ...))
  args.n = 5
  print(select("#", ...))
end
f10(1, 2, 3)
--> =3
--> =5

---------------------------------------------------------------------
-- Invalid n triggers error when ... is expanded
---------------------------------------------------------------------
local function f11(... args)
  args.n = -1
  return ...
end
print(pcall(f11, 1, 2))
--> ~false.*no proper 'n'

local function f12(... args)
  args.n = 1.0
  return ...
end
print(pcall(f12, 1, 2))
--> ~false.*no proper 'n'

---------------------------------------------------------------------
-- Writing to vararg table then returning via ...
---------------------------------------------------------------------
local function f13(a, v, ... t)
  for k, val in pairs(v) do t[k] = val end
  return ...
end
local r1, r2, r3 = f13(10, {11, [3] = 33}, 1, 2, 3)
print(r1, r2, r3)
--> =11	2	33
