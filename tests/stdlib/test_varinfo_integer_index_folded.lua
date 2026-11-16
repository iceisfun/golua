-- Test: t[k] with a key that folds to a small integer is an 'integer index'
-- What: reference luaK_indexed uses GETI for any key that is an integer
-- constant from 0 to 255 once parsed, so errors name the access
-- (field 'integer index'). golua took GETI only for an integer literal; t[K]
-- with an inlined <const> K, t[1+1] or t[nil or 1] went through GETTABLE and
-- were named (field '?').

local function errmsg(f)
  local ok, msg = pcall(f)
  assert(not ok)
  return msg
end

local function check(f, name)
  local msg = errmsg(f)
  assert(msg:find("(field '" .. name .. "')", 1, true), msg)
end

local t = {}
local K0 <const> = 0
local K300 <const> = 300
local KF <const> = 1.0

check(function() return t[0] + 1 end, "integer index")
check(function() return t[K0] + 1 end, "integer index")
check(function() return t[(K0)] + 1 end, "integer index")
check(function() t[1 + 1]() end, "integer index")
check(function() return t[-1 + 2] + 1 end, "integer index")
check(function() t[K0].x = 1 end, "integer index")
check(function() return t[nil or 3] + 1 end, "integer index")
check(function() local u = {} return u[K0 + 5] + 1 end, "integer index")
check(function() return t[255] + 1 end, "integer index")

-- Out of GETI range, or not an integer: a register key, named '?'.
check(function() return t[K300] + 1 end, "?")
check(function() return t[256] + 1 end, "?")
check(function() return t[-1] + 1 end, "?")
check(function() return t[KF] + 1 end, "?")
check(function() return t[1.0] + 1 end, "?")

-- The left operands an and/or key passes over still run.
do
  local n = 0
  local function f() n = n + 1 end
  check(function() return t[(f() or 1) and 2] + 1 end, "integer index")
  check(function() local u = {} return u[(f() or 1) and 2] + 1 end, "integer index")
  assert(n == 2)
  local u = {[1] = "a", [2] = "b"}
  assert(u[K0 + 1] == "a" and u[(f() or 1) and 2] == "b" and n == 3)
end
