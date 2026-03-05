-- ==========================================================================
-- Fengari test extraction: Lua 5.4 conformance: local variables
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: suite_locals
-- Total tests: 5
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- Fengari-specific getenv: get the _ENV upvalue of a function
function getenv(f)
  local i = 1
  while true do
    local name, val = debug.getupvalue(f, i)
    if not name then return nil end
    if name == "_ENV" then return val end
    i = i + 1
  end
end

-- [Test 1] [test-suite] locals: bug in 5.1
-- Verifies: all assert() calls pass without error
do
  local function f(x) x = nil; return x end
  assert(f(10) == nil)

  local function f() local x; return x end
  assert(f(10) == nil)

  local function f(x) x = nil; local y; return x, y end
  assert(f(10) == nil and select(2, f(20)) == nil)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 2] [test-suite] locals: local scope
-- Verifies: all assert() calls pass without error
do
  do
    local i = 10
    do local i = 100; assert(i==100) end
    do local i = 1000; assert(i==1000) end
    assert(i == 10)
    if i ~= 10 then
      local i = 20
    else
      local i = 30
      assert(i == 30)
    end
  end

  f = nil

  local f
  x = 1

  a = nil
  load('local a = {}')()
  assert(a == nil)

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
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 3] [test-suite] locals: test for global table of loaded chunks
-- Verifies: all assert() calls pass without error
do
  assert(getenv(load"a=3") == _G)
  local c = {}; local f = load("a = 3", nil, nil, c)
  assert(getenv(f) == c)
  assert(c.a == nil)
  f()
  assert(c.a == 3)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 4] [test-suite] locals: old test for limits for special instructions (now just a generic test)
-- Verifies: all assert() calls pass without error
do
  do
    local i = 2
    local p = 4    -- p == 2^i
    repeat
      for j=-3,3 do
        assert(load(string.format([[local a=%s;
                                          a=a+%s;
                                          assert(a ==2^%s)]], j, p-j, i), '')) ()
        assert(load(string.format([[local a=%s;
                                          a=a-%s;
                                          assert(a==-2^%s)]], -j, p-j, i), '')) ()
        assert(load(string.format([[local a,b=0,%s;
                                          a=b-%s;
                                          assert(a==-2^%s)]], -j, p-j, i), '')) ()
      end
      p = 2 * p;  i = i + 1
    until p <= 0
  end
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 5] [test-suite] locals: testing lexical environments
-- Verifies: all assert() calls pass without error
do
  assert(_ENV == _G)

  do
    local dummy
    local _ENV = (function (...) return ... end)(_G, dummy)   -- {

    do local _ENV = {assert=assert}; assert(true) end
    mt = {_G = _G}
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
  end
  print("PASS")
end
--> =PASS
