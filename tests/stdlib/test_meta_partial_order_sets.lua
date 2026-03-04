-- Test: events.lua - Partial order with sets
-- From: events.lua
-- What: Tests using metamethods to implement set comparison (partial order)

do
  local t = {}

  local function rawSet(x)
    local y = {}
    for _,k in pairs(x) do y[k] = 1 end
    return y
  end

  local function Set(x)
    return setmetatable(rawSet(x), t)
  end

  t.__lt = function (a,b)
    for k in pairs(a) do
      if not b[k] then return false end
      b[k] = nil
    end
    return next(b) ~= nil
  end

  t.__le = function (a,b)
    for k in pairs(a) do
      if not b[k] then return false end
    end
    return true
  end

  assert(Set{1,2,3} < Set{1,2,3,4})
  assert(not(Set{1,2,3,4} < Set{1,2,3,4}))
  assert((Set{1,2,3,4} <= Set{1,2,3,4}))
  assert((Set{1,2,3,4} >= Set{1,2,3,4}))
  assert(not (Set{1,3} <= Set{3,5}))
  assert(not(Set{1,3} <= Set{3,5}))
  assert(not(Set{1,3} >= Set{3,5}))

  t.__eq = function (a,b)
    for k in pairs(a) do
      if not b[k] then return false end
      b[k] = nil
    end
    return next(b) == nil
  end

  local s = Set{1,3,5}
  assert(s == Set{3,5,1})
  assert(not rawequal(s, Set{3,5,1}))
  assert(rawequal(s, s))
  assert(Set{1,3,5,1} == rawSet{3,5,1})
  assert(rawSet{1,3,5,1} == Set{3,5,1})
  assert(Set{1,3,5} ~= Set{3,5,1,6})

  -- '__eq' is not used for table accesses
  t[Set{1,3,5}] = 1
  assert(t[Set{1,3,5}] == nil)
end
