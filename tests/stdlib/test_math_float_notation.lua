-- Test: math.lua - Basic float notation and string-to-number coercion
-- From: math.lua
-- What: Tests float literal syntax, string arithmetic coercion, and minus-zero table indexing.

do
  -- basic float notation
  assert(0e12 == 0 and .0 == 0 and 0. == 0 and .2e2 == 20 and 2.E-1 == 0.2)

  do
    local a,b,c = "2", " 3e0 ", " 10  "
    assert(a+b == 5 and -b == -3 and b+"2" == 5 and "10"-c == 0)
    assert(type(a) == 'string' and type(b) == 'string' and type(c) == 'string')
    assert(a == "2" and b == " 3e0 " and c == " 10  " and -c == -"  10 ")
    assert(c%a == 0 and a^b == 08)
    a = 0
    assert(a == -a and 0 == -0)
  end

  do
    local x = -1
    local mz = 0/x   -- minus zero
    local t = {[0] = 10, 20, 30, 40, 50}
    assert(t[mz] == t[0] and t[-0] == t[0])
  end
end
