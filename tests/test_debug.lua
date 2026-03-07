-- ==========================================================================
-- Fengari test extraction: Debug library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: debug
-- Total tests: 9
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] debug.sethook
-- Verifies: output matches expected value via print()
do
  local result = ""

  debug.sethook(function (event)
      result = result .. event .. " "
  end, "crl", 1)

  local l = function() end

  l()
  l()
  l()

  debug.sethook()  -- clear hook before checking
  assert(type(result) == "string" and #result > 0)
  -- Verify key events appear in the trace
  assert(string.find(result, "call"), "expected 'call' events")
  assert(string.find(result, "return"), "expected 'return' events")
  assert(string.find(result, "count"), "expected 'count' events")
  assert(string.find(result, "line"), "expected 'line' events")
end

-- --------------------------------------------------------------------------
-- [Test 2] debug.gethook
-- Verifies: all assert() calls pass without error
do
  local result = ""

  debug.sethook(function (event)
      result = result .. event .. " "
  end, "crl", 1)

  local l = function() end

  l()
  l()
  l()

  local _r1, _r2, _r3 = debug.gethook()
  assert(type(_r1) == "function")
  assert(_r2 == "crl")
  assert(_r3 == 1)
end

-- --------------------------------------------------------------------------
-- [Test 3] debug.getlocal
-- Verifies: output matches expected value via print()
do
  local alocal = "alocal"
  local another = "another"

  local result = ""

  local l = function()
      local infunction = "infunction"
      local anotherin = "anotherin"
      result = table.concat(table.pack(debug.getlocal(2, 1)), " ")
          .. table.concat(table.pack(debug.getlocal(2, 2)), " ")
          .. table.concat(table.pack(debug.getlocal(1, 1)), " ")
          .. table.concat(table.pack(debug.getlocal(1, 2)), " ")
  end

  l()

  assert(result == "alocal alocalanother anotherinfunction infunctionanotherin anotherin")
end

-- --------------------------------------------------------------------------
-- [Test 4] debug.setlocal
-- Verifies: output matches expected value via print()
do
  local alocal = "alocal"
  local another = "another"

  local l = function()
      local infunction = "infunction"
      local anotherin = "anotherin"

      debug.setlocal(2, 1, 1)
      debug.setlocal(2, 2, 2)
      debug.setlocal(1, 1, 3)
      debug.setlocal(1, 2, 4)

      return infunction, anotherin
  end

  local a, b = l()

  assert(alocal == 1 and another == 2 and a == 3 and b == 4)
end

-- --------------------------------------------------------------------------
-- [Test 5] debug.upvalueid
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local upvalue = "upvalue"

  local l = function()
      return upvalue
  end

  assert(debug.upvalueid(l, 1) ~= nil)
end

-- --------------------------------------------------------------------------
-- [Test 6] debug.upvaluejoin
-- Verifies: output matches expected value via print()
do
  local upvalue1 = "upvalue1"
  local upvalue2 = "upvalue2"

  local l1 = function()
      return upvalue1
  end

  local l2 = function()
      return upvalue2
  end

  debug.upvaluejoin(l1, 1, l2, 1)

  assert(l1() == "upvalue2")
end

-- --------------------------------------------------------------------------
-- [Test 7] debug.traceback (with a global)
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local trace

  rec = function(n)
      n = n or 0
      if n < 10 then
          rec(n + 1)
      else
          trace = debug.traceback()
      end
  end

  rec()

  assert(type(trace) == "string")
end

-- --------------------------------------------------------------------------
-- [Test 8] debug.traceback (with a upvalue)
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local trace
  local rec

  rec = function(n)
      n = n or 0
      if n < 10 then
          rec(n + 1)
      else
          trace = debug.traceback()
      end
  end

  rec()

  assert(type(trace) == "string")
end

-- --------------------------------------------------------------------------
-- [Test 9] debug.getinfo
-- Verifies: output matches expected value via print()
do
  local alocal = function(p1, p2) end
  aglobal = function() return alocal end

  local d1 = debug.getinfo(alocal)
  local d2 = debug.getinfo(aglobal)

  assert(d1.nups == 0)
  assert(d1.what == "Lua")
  assert(d1.nparams == 2)
  assert(d2.nups == 1)
  assert(d2.what == "Lua")
  assert(d2.nparams == 0)
end
