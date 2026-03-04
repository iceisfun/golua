-- Test: events.lua - Comparison metamethods (__lt, __le, __eq)
-- From: events.lua
-- What: Tests comparison metamethods with various orderings and edge cases

do
  local t = {}

  t.__lt = function (a,b,c)
    collectgarbage()
    assert(c == nil)
    if type(a) == 'table' then a = a.x end
    if type(b) == 'table' then b = b.x end
    return a<b, "dummy"
  end

  t.__le = function (a,b,c)
    assert(c == nil)
    if type(a) == 'table' then a = a.x end
    if type(b) == 'table' then b = b.x end
    return a<=b, "dummy"
  end

  t.__eq = function (a,b,c)
    assert(c == nil)
    if type(a) == 'table' then a = a.x end
    if type(b) == 'table' then b = b.x end
    return a == b, "dummy"
  end

  local function Op(x) return setmetatable({x=x}, t) end

  local function test (a, b, c)
    assert(not(Op(1)<Op(1)) and (Op(1)<Op(2)) and not(Op(2)<Op(1)))
    assert(not(1 < Op(1)) and (Op(1) < 2) and not(2 < Op(1)))
    assert(not(Op('a')<Op('a')) and (Op('a')<Op('b')) and not(Op('b')<Op('a')))
    assert(not('a' < Op('a')) and (Op('a') < 'b') and not(Op('b') < Op('a')))
    assert((Op(1)<=Op(1)) and (Op(1)<=Op(2)) and not(Op(2)<=Op(1)))
    assert((Op('a')<=Op('a')) and (Op('a')<=Op('b')) and not(Op('b')<=Op('a')))
    assert(not(Op(1)>Op(1)) and not(Op(1)>Op(2)) and (Op(2)>Op(1)))
    assert(not(Op('a')>Op('a')) and not(Op('a')>Op('b')) and (Op('b')>Op('a')))
    assert((Op(1)>=Op(1)) and not(Op(1)>=Op(2)) and (Op(2)>=Op(1)))
    assert((1 >= Op(1)) and not(1 >= Op(2)) and (Op(2) >= 1))
    assert((Op('a')>=Op('a')) and not(Op('a')>=Op('b')) and (Op('b')>=Op('a')))
    assert(('a' >= Op('a')) and not(Op('a') >= 'b') and (Op('b') >= Op('a')))
    assert(Op(1) == Op(1) and Op(1) ~= Op(2))
    assert(Op('a') == Op('a') and Op('a') ~= Op('b'))
    assert(a == a and a ~= b)
    assert(Op(3) == c)
  end

  test(Op(1), Op(2), Op(3))
end
