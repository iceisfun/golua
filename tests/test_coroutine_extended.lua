-- ==========================================================================
-- Fengari test extraction: Coroutine library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: coroutine
-- Total tests: 5
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.

-- [Test 1] coroutine.create, coroutine.yield, coroutine.resume
do
  local co = coroutine.create(function (start)
      local b = coroutine.yield(start * start);
      coroutine.yield(b * b)
  end)

  local success, pow = coroutine.resume(co, 5)
  success, pow = coroutine.resume(co, pow)

  assert(pow == 625)
end

-- --------------------------------------------------------------------------
-- [Test 2] coroutine.status
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

  assert(s1 == "suspended")
  assert(s2 == "dead")
end

-- --------------------------------------------------------------------------
-- [Test 3] coroutine.isyieldable
do
  local co = coroutine.create(function ()
      coroutine.yield(coroutine.isyieldable());
  end)

  local success, yieldable = coroutine.resume(co)

  assert(yieldable == true)
  assert(coroutine.isyieldable() == false)
end

-- --------------------------------------------------------------------------
-- [Test 4] coroutine.running
-- Inside a coroutine, running() returns the thread and false (not main)
do
  local running, ismain

  local co = coroutine.create(function ()
      running, ismain = coroutine.running()
  end)

  coroutine.resume(co)

  assert(type(running) == "thread")
  assert(ismain == false)
end

-- --------------------------------------------------------------------------
-- [Test 5] coroutine.wrap
do
  local co = coroutine.wrap(function (start)
      local b = coroutine.yield(start * start);
      coroutine.yield(b * b)
  end)

  local pow = co(5)
  pow = co(pow)

  assert(pow == 625)
end
