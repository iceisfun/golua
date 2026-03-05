-- ==========================================================================
-- Fengari test extraction: VM opcodes and core execution
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: vm
-- Total tests: 28
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] LOADK, RETURN
-- Verifies: output matches expected value via print()
do
  local a = "hello world"
  print(a)
end
--> =hello world

-- --------------------------------------------------------------------------
-- [Test 2] MOVE
-- Verifies: output matches expected value via print()
do
  local a = "hello world"
  local b = a
  print(b)
end
--> =hello world

-- --------------------------------------------------------------------------
-- [Test 3] Binary op
-- Verifies: output matches expected value via print()
do
  local a = 5
  local b = 10
  print(a + b, a - b, a * b, a / b, a % b, a^b, a // b, a & b, a | b, a ~ b, a << b, a >> b)
end
--> =15	-5	50	0.5	5	9765625.0	0	0	15	15	5120	0

-- --------------------------------------------------------------------------
-- [Test 4] Unary op, LOADBOOL
-- Verifies: output matches expected value via print()
do
  local a = 5
  local b = false
  print(-a, not b, ~a)
end
--> =-5	true	-6

-- --------------------------------------------------------------------------
-- [Test 5] NEWTABLE
-- Verifies: all assert() calls pass without error
do
  local a = {}
  local _r1 = a
  assert(type(_r1) == "table")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 6] CALL
-- Verifies: output matches expected value via print()
do
  local f = function (a, b)
      return a + b
  end

  local c = f(1, 2)

  print(c)
end
--> =3

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

  print(c, d, e)
end
--> =3	-1	2

-- --------------------------------------------------------------------------
-- [Test 8] TAILCALL
-- Verifies: output matches expected value via print()
do
  local f = function (a, b)
      return a + b
  end

  print(f(1,2))
end
--> =3

-- --------------------------------------------------------------------------
-- [Test 9] VARARG
-- Verifies: output matches expected value via print()
do
  local f = function (...)
      return ...
  end

  print(f(1,2,3))
end
--> =1	2	3

-- --------------------------------------------------------------------------
-- [Test 10] LE, JMP
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  print(a >= b)
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 11] LT
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  print(a > b)
end
--> =false

-- --------------------------------------------------------------------------
-- [Test 12] EQ
-- Verifies: output matches expected value via print()
do
  local a, b = 1, 1

  print(a == b)
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 13] TESTSET (and)
-- Verifies: output matches expected value via print()
do
  local a = true
  local b = "hello"

  print(a and b)
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 14] TESTSET (or)
-- Verifies: output matches expected value via print()
do
  local a = false
  local b = "hello"

  print(a or b)
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 15] TEST (true)
-- Verifies: output matches expected value via print()
do
  local a = true
  local b = "hello"

  if a then
      return b
  end

  print("goodbye")
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 16] TEST (false)
-- Verifies: output matches expected value via print()
do
  local a = false
  local b = "hello"

  if a then
      return b
  end

  print("goodbye")
end
--> =goodbye

-- --------------------------------------------------------------------------
-- [Test 17] FORPREP, FORLOOP (int)
-- Verifies: output matches expected value via print()
do
  local total = 0

  for i = 0, 10 do
      total = total + i
  end

  print(total)
end
--> =55

-- --------------------------------------------------------------------------
-- [Test 18] FORPREP, FORLOOP (float)
-- Verifies: output matches expected value via print()
do
  local total = 0

  for i = 0.5, 10.5 do
      total = total + i
  end

  print(total)
end
--> =60.5

-- --------------------------------------------------------------------------
-- [Test 19] SETTABLE, GETTABLE
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {}

  t[1] = "hello"
  t["two"] = "world"

  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 20] SETUPVAL, GETUPVAL
-- Verifies: output matches expected value via print()
do
  local up = "hello"

  local f = function ()
      upup = "yo"
      up = "world"
      return up;
  end

  print(f())
end
--> =world

-- --------------------------------------------------------------------------
-- [Test 21] SETTABUP, GETTABUP
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  t = {}

  t[1] = "hello"
  t["two"] = "world"

  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 22] SELF
-- Verifies: output matches expected value via print()
do
  local t = {}

  t.value = "hello"
  t.get = function (self)
      return self.value
  end

  print(t:get())
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 23] SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 4, 5, 6, 7, 8, 9}

  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 24] Variable SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local a = function ()
      return 6, 7, 8, 9
  end

  local t = {1, 2, 3, 4, 5, a()}

  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 25] Long SETLIST
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5, 1, 2, 3, 4, 5}

  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 26] TFORCALL, TFORLOOP
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

  print(r)
end
--> =6

-- --------------------------------------------------------------------------
-- [Test 27] LEN
-- Verifies: output matches expected value via print()
do
  local t = {[10000] = "foo"}
  local t2 = {1, 2, 3}
  local s = "hello"

  print(#t, #t2, #s)
end
--> =0	3	5

-- --------------------------------------------------------------------------
-- [Test 28] CONCAT
-- Verifies: output matches expected value via print()
do
  print("hello " .. 2 .. " you")
end
--> =hello 2 you
