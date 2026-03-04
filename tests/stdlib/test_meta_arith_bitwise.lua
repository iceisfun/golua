-- Test: events.lua - All arithmetic/bitwise metamethods
-- From: events.lua
-- What: Tests all arithmetic (__add, __sub, __mul, __div, __idiv, __mod, __pow, __unm),
--       bitwise (__band, __bor, __bxor, __shl, __shr, __bnot), and length (__len)
--       metamethods with capture verification

do
  local cap
  local t = {}
  local a = setmetatable({}, t)
  local b = setmetatable({}, t)
  setmetatable(b, t)

  local function f(op)
    return function (...) cap = {[0] = op, ...} ; return (...) end
  end

  t.__add = f("add")
  t.__sub = f("sub")
  t.__mul = f("mul")
  t.__div = f("div")
  t.__idiv = f("idiv")
  t.__mod = f("mod")
  t.__unm = f("unm")
  t.__pow = f("pow")
  t.__len = f("len")
  t.__band = f("band")
  t.__bor = f("bor")
  t.__bxor = f("bxor")
  t.__shl = f("shl")
  t.__shr = f("shr")
  t.__bnot = f("bnot")
  t.__lt = f("lt")
  t.__le = f("le")

  local function checkcap (t)
    assert(#cap + 1 == #t)
    for i = 1, #t do
      assert(cap[i - 1] == t[i])
      assert(math.type(cap[i - 1]) == math.type(t[i]))
    end
  end

  assert(b+5 == b); checkcap{"add", b, 5}
  assert(5.2 + b == 5.2); checkcap{"add", 5.2, b}
  assert(b+'5' == b); checkcap{"add", b, '5'}
  assert(5+b == 5); checkcap{"add", 5, b}
  assert('5'+b == '5'); checkcap{"add", '5', b}
  b=b-3; assert(getmetatable(b) == t); checkcap{"sub", b, 3}
  assert(5-a == 5); checkcap{"sub", 5, a}
  assert('5'-a == '5'); checkcap{"sub", '5', a}
  assert(a*a == a); checkcap{"mul", a, a}
  assert(a/0 == a); checkcap{"div", a, 0}
  assert(a/0.0 == a); checkcap{"div", a, 0.0}
  assert(a%2 == a); checkcap{"mod", a, 2}
  assert(a // (1/0) == a); checkcap{"idiv", a, 1/0}
  ;(function () assert(a & "hi" == a) end)(); checkcap{"band", a, "hi"}
  ;(function () assert(10 & a  == 10) end)(); checkcap{"band", 10, a}
  ;(function () assert(a | 10  == a) end)(); checkcap{"bor", a, 10}
  assert(a | "hi" == a); checkcap{"bor", a, "hi"}
  assert("hi" ~ a == "hi"); checkcap{"bxor", "hi", a}
  ;(function () assert(10 ~ a == 10) end)(); checkcap{"bxor", 10, a}
  assert(-a == a); checkcap{"unm", a, a}
  assert(a^4.0 == a); checkcap{"pow", a, 4.0}
  assert(a^'4' == a); checkcap{"pow", a, '4'}
  assert(4^a == 4); checkcap{"pow", 4, a}
  assert('4'^a == '4'); checkcap{"pow", '4', a}
  assert(#a == a); checkcap{"len", a, a}
  assert(~a == a); checkcap{"bnot", a, a}
  assert(a << 3 == a); checkcap{"shl", a, 3}
  assert(1.5 >> a == 1.5); checkcap{"shr", 1.5, a}

  -- for comparison operators, all results are true
  assert(5.0 > a); checkcap{"lt", a, 5.0}
  assert(a >= 10); checkcap{"le", 10, a}
  assert(a <= -10.0); checkcap{"le", a, -10.0}
  assert(a < -10); checkcap{"lt", a, -10}
end
