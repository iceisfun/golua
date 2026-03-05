-- ==========================================================================
-- Fengari test extraction: Base library functions (print, type, pcall, etc.)
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: baselib
-- Total tests: 17
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] print
-- Verifies: code executes without runtime error
-- NOTE: Run-only test (verifies no runtime error)
do
  print("hello", "world", 123)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 2] setmetatable, getmetatable
-- Verifies: all assert() calls pass without error
do
  local mt = {
      __index = function ()
          return "hello"
      end
  }

  local t = {}

  setmetatable(t, mt);

  local _r1, _r2 = t[1], getmetatable(t)
  assert(_r1 == "hello")
  assert(type(_r2) == "table")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 3] rawequal
-- Verifies: output matches expected value via print()
do
  local mt = {
      __eq = function ()
          return true
      end
  }

  local t1 = {}
  local t2 = {}

  setmetatable(t1, mt);

  print(rawequal(t1, t2), t1 == t2)
end
--> =false	true

-- --------------------------------------------------------------------------
-- [Test 4] rawset, rawget
-- Verifies: output matches expected value via print()
do
  local mt = {
      __newindex = function (table, key, value)
          rawset(table, key, "hello")
      end
  }

  local t = {}

  setmetatable(t, mt);

  t["yo"] = "bye"
  rawset(t, "yoyo", "bye")

  print(rawget(t, "yo"), t["yo"], rawget(t, "yoyo"), t["yoyo"])
end
--> =hello	hello	bye	bye

-- --------------------------------------------------------------------------
-- [Test 5] type
-- Verifies: output matches expected value via print()
do
  print(type(1), type(true), type("hello"), type({}), type(nil))
end
--> =number	boolean	string	table	nil

-- --------------------------------------------------------------------------
-- [Test 6] error
-- Extraction error: these tests were meant to be inside pcall. Fixed.
do
  local ok, msg = pcall(error, "you fucked up")
  assert(not ok and string.find(msg, "you fucked up"))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 7] error, protected
do
  local ok, msg = pcall(error, "you fucked up")
  assert(not ok and string.find(msg, "you fucked up"))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 8] pcall
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local willFail = function ()
      error("you fucked up")
  end

  print(pcall(willFail))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 9] xpcall
-- Verifies: all assert() calls pass without error
do
  local willFail = function ()
      error("you fucked up")
  end

  local msgh = function (err)
      return "Something's wrong: " .. err
  end

  local ok, msg = xpcall(willFail, msgh)
  assert(ok == false)
  assert(type(msg) == "string" and string.find(msg, "Something's wrong"))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 10] ipairs
-- Verifies: output matches expected value via print()
do
  local t = {1, 2, 3, 4, 5, ['yo'] = 'lo'}

  local sum = 0
  for i, v in ipairs(t) do
      sum = sum + v
  end

  print(sum)
end
--> =15

-- --------------------------------------------------------------------------
-- [Test 11] select
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print({select('#', 1, 2, 3)}, {select(2, 1, 2, 3)}, {select(-2, 1, 2, 3)})
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 12] tonumber
-- Verifies: all assert() calls pass without error
do
  local _r1, _r2, _r3, _r4, _r5 = tonumber('foo'), tonumber('123'), tonumber('12.3'), tonumber('az', 36), tonumber('10', 2)
  assert(_r1 == nil)
  assert(_r2 == 123)
  assert(_r3 == 12.3)
  assert(_r4 == 395)
  assert(_r5 == 2)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 13] assert (extraction error: original tested pcall(assert, 1<0, msg))
do
  local ok, msg = pcall(assert, 1 < 0, "this doesn't makes sense")
  assert(not ok and string.find(msg, "this doesn't makes sense"))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 14] rawlen
-- Verifies: output matches expected value via print()
do
  print(rawlen({1, 2, 3}), rawlen('hello'))
end
--> =3	5

-- --------------------------------------------------------------------------
-- [Test 15] next
-- Verifies: output matches expected value via print()
do
  local total = 0
  local t = {
      1,
      two = 2,
      3,
      four = 4
  }

  for k,v in next, t, nil do
      total = total + v
  end

  print(total)
end
--> =10

-- --------------------------------------------------------------------------
-- [Test 16] pairs
-- Verifies: output matches expected value via print()
do
  local total = 0
  local t = {
      1,
      two = 2,
      3,
      four = 4
  }

  for k,v in pairs(t) do
      total = total + v
  end

  print(total)
end
--> =10

-- --------------------------------------------------------------------------
-- [Test 17] pairs with __pairs
-- Verifies: output matches expected value via print()
do
  local total = 0

  local mt = {
      __pairs = function(t)
          return next, {5, 6, 7, 8}, nil
      end
  }

  local t = {
      1,
      two = 2,
      3,
      four = 4
  }

  setmetatable(t, mt)

  for k,v in pairs(t) do
      total = total + v
  end

  print(total)
end
--> =26
