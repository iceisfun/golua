-- Test: Lexical environments (_ENV)
-- From: locals.lua
-- What: Tests lexical environments: overriding _ENV in nested scopes, using custom environment tables for function definitions, and verifying that _ENV correctly controls name resolution.

do
  local function getenv (f)
    local a,b = debug.getupvalue(f, 1)
    assert(a == '_ENV')
    return b
  end

  assert(_ENV == _G)

  do
  local dummy
  local _ENV = (function (...) return ... end)(_G, dummy)   -- {

  do local _ENV = {assert=assert}; assert(true) end
  local mt = {_G = _G}
  local foo,x
  A = false    -- "declare" A
  do local _ENV = mt
    function foo (x)
      A = x
      do local _ENV =  _G; A = 1000 end
      return function (x) return A .. x end
    end
  end
  assert(getenv(foo) == mt)
  x = foo('hi'); assert(mt.A == 'hi' and A == 1000)
  assert(x('*') == mt.A .. '*')

  do local _ENV = {assert=assert, A=10};
    do local _ENV = {assert=assert, A=20};
      assert(A==20);x=A
    end
    assert(A==10 and x==20)
  end
  assert(x==20)

  A = nil
  end
end
