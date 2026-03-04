-- Test: constructs.lua - Multiple return values in expressions
-- From: constructs.lua
-- What: Tests how multiple return values interact with expressions and parenthesization

do
  function f () return 1,2,3; end
  local a, b, c = f();
  assert(a==1 and b==2 and c==3)
  a, b, c = (f());
  assert(a==1 and b==nil and c==nil)
end
