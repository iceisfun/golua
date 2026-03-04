-- Test: Local variable assignment and shadowing in functions
-- From: locals.lua
-- What: Tests local variable declarations, multiple assignment, and shadowing within function bodies and if/else blocks.

do
  local f
  local x = 1

  function f (a)
    local _1, _2, _3, _4, _5
    local _6, _7, _8, _9, _10
    local x = 3
    local b = a
    local c,d = a,b
    if (d == b) then
      local x = 'q'
      x = b
      assert(x == 2)
    else
      assert(nil)
    end
    assert(x == 3)
    local f = 10
  end

  local b=10
  local a; repeat local b; a,b=1,2; assert(a+1==b); until a+b==3

  assert(x == 1)

  f(2)
  assert(type(f) == 'function')
end
