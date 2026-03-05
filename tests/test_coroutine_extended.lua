-- ==========================================================================
-- Fengari test extraction: Coroutine library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: coroutine
-- Total tests: 5
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] coroutine.create, coroutine.yield, coroutine.resume
-- Verifies: output matches expected value via print()
do
  local co = coroutine.create(function (start)
      local b = coroutine.yield(start * start);
      coroutine.yield(b * b)
  end)

  local success, pow = coroutine.resume(co, 5)
  success, pow = coroutine.resume(co, pow)

  print(pow)
end
--> =625

-- --------------------------------------------------------------------------
-- [Test 2] coroutine.status
-- Verifies: output matches expected value via print()
do
  local co = coroutine.create(function (start)
      local b = coroutine.yield(start * start);
      coroutine.yield(b * b)
  end)

  local s1 = coroutine.status(co)

  local success, pow = coroutine.resume(co, 5)
  success, pow = coroutine.resume(co, pow)

  coroutine.resume(co, pow)

  local s2 = coroutine.status(co)

  print(s1, s2)
end
--> =suspended	dead

-- --------------------------------------------------------------------------
-- [Test 3] coroutine.isyieldable
-- Verifies: output matches expected value via print()
do
  local co = coroutine.create(function ()
      coroutine.yield(coroutine.isyieldable());
  end)

  local succes, yieldable = coroutine.resume(co)

  print(yieldable, coroutine.isyieldable())
end
--> =true	false

-- --------------------------------------------------------------------------
-- [Test 4] coroutine.running
-- Verifies: output matches expected value via print()
do
  local running, ismain

  local co = coroutine.create(function ()
      running, ismain = coroutine.running()
  end)

  coroutine.resume(co)

  print(running, ismain)
end
--> =false

-- --------------------------------------------------------------------------
-- [Test 5] coroutine.wrap
-- Verifies: output matches expected value via print()
do
  local co = coroutine.wrap(function (start)
      local b = coroutine.yield(start * start);
      coroutine.yield(b * b)
  end)

  pow = co(5)
  pow = co(pow)

  print(pow)
end
--> =625
