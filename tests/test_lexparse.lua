-- ==========================================================================
-- Fengari test extraction: Lexer and parser edge cases
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: lexparse
-- Total tests: 27
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] LOADK, RETURN
-- Verifies: output matches expected value via print()
do
  local a = "hello world"
  assert(a == "hello world")
end

-- --------------------------------------------------------------------------
-- [Test 2] MOVE
-- Verifies: output matches expected value via print()
do
  local a = "hello world"
  local b = a
  assert(b == "hello world")
end

-- --------------------------------------------------------------------------
-- [Test 3] Binary op
-- Verifies: output matches expected value via print()
do
  local a = 5
  local b = 10
  assert(a + b == 15)
  assert(a - b == -5)
  assert(a * b == 50)
  assert(a / b == 0.5)
  assert(a % b == 5)
  assert(a^b == 9765625.0)
  assert(a // b == 0)
  assert(a & b == 0)
  assert(a | b == 15)
  assert(a ~ b == 15)
  assert(a << b == 5120)
  assert(a >> b == 0)
end

-- --------------------------------------------------------------------------
-- [Test 4] Unary op, LOADBOOL
-- Verifies: output matches expected value via print()
do
  local a = 5
  local b = false
  assert(-a == -5)
  assert((not b) == true)
  assert(~a == -6)
end

-- --------------------------------------------------------------------------
-- [Test 5] NEWTABLE
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local a = {}
  assert(type(a) == "table")
end

-- --------------------------------------------------------------------------
-- [Test 6] CALL
-- Verifies: output matches expected value via print()
do
  local f = function (a, b)
      return a + b
  end

  local c = f(1, 2)

  assert(c == 3)
end

-- --------------------------------------------------------------------------
-- [Test 7] Multiple return
-- Verifies: output matches expected value via print()
do
  local f = function (a, b)
      return a + b, a - b, a * b
  end

  local c
  local d
  local e

  c, d, e = f(1,2)

  assert(c == 3 and d == -1 and e == 2)
end

-- --------------------------------------------------------------------------
-- [Test 8] TAILCALL
-- Verifies: output matches expected value via print()
do
  local f = function (a, b)
      return a + b
  end

  assert(f(1,2) == 3)
end

-- --------------------------------------------------------------------------
-- [Test 9] VARARG
-- Verifies: output matches expected value via print()
do
  local f = function (...)
      return ...
  end

  local a, b, c = f(1,2,3)
  assert(a == 1 and b == 2 and c == 3)
end

-- --------------------------------------------------------------------------
-- [Test 10] LE, JMP
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  assert(a >= b)
end

-- --------------------------------------------------------------------------
-- [Test 11] LT
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  assert(not (a > b))
end

-- --------------------------------------------------------------------------
-- [Test 12] EQ
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  assert(a == b)
end

-- --------------------------------------------------------------------------
-- [Test 13] TESTSET (and)
-- Verifies: output matches expected value via print()
do
  local a = true
  local b = "hello"

  assert((a and b) == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 14] TESTSET (or)
-- Verifies: output matches expected value via print()
do
  local a = false
  local b = "hello"

  assert((a or b) == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 15] TEST (false)
-- Verifies: if-branch is skipped when condition is false
do
  local a = false
  local b = "hello"
  local result = "goodbye"

  if a then
      result = b
  end

  assert(result == "goodbye")
end

-- --------------------------------------------------------------------------
-- [Test 16] FORPREP, FORLOOP (int)
-- Verifies: output matches expected value via print()
do
  local total = 0

  for i = 0, 10 do
      total = total + i
  end

  assert(total == 55)
end

-- --------------------------------------------------------------------------
-- [Test 17] FORPREP, FORLOOP (float)
-- Verifies: output matches expected value via print()
do
  local total = 0

  for i = 0.5, 10.5 do
      total = total + i
  end

  assert(total == 60.5)
end

-- --------------------------------------------------------------------------
-- [Test 18] SETTABLE, GETTABLE
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {}

  t[1] = "hello"
  t["two"] = "world"

  assert(t[1] == "hello")
  assert(t["two"] == "world")
end

-- --------------------------------------------------------------------------
-- [Test 19] SETUPVAL, GETUPVAL
-- Verifies: output matches expected value via print()
do
  local up = "hello"

  local f = function ()
      upup = "yo"
      up = "world"
      return up;
  end

  assert(f() == "world")
end

-- --------------------------------------------------------------------------
-- [Test 20] SETTABUP, GETTABUP
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  t = {}

  t[1] = "hello"
  t["two"] = "world"

  assert(t[1] == "hello")
  assert(t["two"] == "world")
end

-- --------------------------------------------------------------------------
-- [Test 21] SELF
-- Verifies: output matches expected value via print()
do
  local t = {}

  t.value = "hello"
  t.get = function (self)
      return self.value
  end

  assert(t:get() == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 22] SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 4, 5, 6, 7, 8, 9}

  assert(#t == 9)
  assert(t[1] == 1 and t[9] == 9)
end

-- --------------------------------------------------------------------------
-- [Test 23] Variable SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local a = function ()
      return 6, 7, 8, 9
  end

  local t = {1, 2, 3, 4, 5, a()}

  assert(#t == 9)
  assert(t[6] == 6 and t[9] == 9)
end

-- --------------------------------------------------------------------------
-- [Test 24] Long SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}

  assert(#t == 75)
end

-- --------------------------------------------------------------------------
-- [Test 25] TFORCALL, TFORLOOP
-- Verifies: output matches expected value via print()
do
  local iterator = function (t, i)
      i = i + 1
      local v = t[i]
      if v then
          return i, v
      end
  end

  local iprs = function(t)
      return iterator, t, 0
  end

  local t = {1, 2, 3}
  local r = 0
  for k,v in iprs(t) do
      r = r + v
  end

  assert(r == 6)
end

-- --------------------------------------------------------------------------
-- [Test 26] LEN
-- Verifies: output matches expected value via print()
do
  local t = {[10000] = "foo"}
  local t2 = {1, 2, 3}
  local s = "hello"

  assert(#t == 0)
  assert(#t2 == 3)
  assert(#s == 5)
end

-- --------------------------------------------------------------------------
-- [Test 27] CONCAT
-- Verifies: output matches expected value via print()
do
  assert("hello " .. 2 .. " you" == "hello 2 you")
end
