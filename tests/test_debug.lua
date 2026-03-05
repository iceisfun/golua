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

  print(result)
end
--> =return count line count line count line call count line return count line count line call count line return count line count line call count line return count line 

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
  print("PASS")
end
--> =PASS

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

  print(result)
end
--> =alocal alocalanother anotherinfunction infunctionanotherin anotherin

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

  print(alocal, another, a, b)
end
--> =1	2	3	4

-- --------------------------------------------------------------------------
-- [Test 5] debug.upvalueid
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local upvalue = "upvalue"

  local l = function()
      return upvalue
  end

  print(debug.upvalueid(l, 1))
  print("PASS")
end
--> =PASS

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

  print(l1())
end
--> =upvalue2

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

  print(trace)
  print("PASS")
end
--> =PASS

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

  print(trace)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 9] debug.getinfo
-- Verifies: output matches expected value via print()
do
  local alocal = function(p1, p2) end
  aglobal = function() return alocal end

  local d1 = debug.getinfo(alocal)
  local d2 = debug.getinfo(aglobal)

  print(d1.short_src, d1.nups, d1.what, d1.nparams, d2.short_src, d2.nups, d2.what, d2.nparams)
end
--> =0	2	1	0
