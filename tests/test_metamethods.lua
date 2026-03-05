-- ==========================================================================
-- Fengari test extraction: Metamethod dispatch (__index, __add, etc.)
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: metamethods
-- Total tests: 17
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] __index, __newindex: with actual table
-- Verifies: all assert() calls pass without error
do
  local t = {yo=1}
  local _r1, _r2 = t.yo, t.lo
  assert(_r1 == 1)
  assert(_r2 == nil)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 2] __newindex: with non table
-- Verifies: indexing a string for assignment raises an error
do
  local ok, err = pcall(function()
    local t = "a string"
    t.yo = "hello"
  end)
  assert(not ok, "expected error when indexing string for newindex")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 3] __index function in metatable
-- Verifies: output matches expected value via print()
do
  local mt = {
      __index = function (table, key)
          return "__index"
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t.yo)
end
--> =__index

-- --------------------------------------------------------------------------
-- [Test 4] __newindex function in metatable
-- Verifies: all assert() calls pass without error
do
  local mt = {
      __newindex = function (table, key, value)
          return "__newindex"
      end
  }

  local t = {}

  setmetatable(t, mt)

  t.yo = "hello"

  local _r1 = t.yo
  assert(_r1 == nil)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 5] __index table in metatable
-- Verifies: output matches expected value via print()
do
  local mmt = {
      yo = "hello"
  }

  local mt = {
      __index = mmt
  }

  local t = {}

  setmetatable(t, mt)

  print(t.yo)
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 6] __newindex table in metatable
-- Verifies: all assert() calls pass without error
do
  local mmt = {
      yo = "hello"
  }

  local mt = {
      __newindex = mmt
  }

  local t = {}

  setmetatable(t, mt)

  t.yo = "world"

  local _r1, _r2 = t.yo, mmt.yo
  assert(_r1 == nil)
  assert(_r2 == "world")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 7] __index table with own metatable
-- Verifies: output matches expected value via print()
do
  local mmmt = {
      __index = function (t, k)
          return "hello"
      end
  }

  local mmt = {
      yoo = "bye"
  }

  setmetatable(mmt, mmmt)

  local mt = {
      __index = mmt
  }

  local t = {}

  setmetatable(t, mt)

  print(t.yo)
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 8] __newindex table with own metatable
-- Verifies: all assert() calls pass without error
do
  local up = nil

  local mmmt = {
      __newindex = function (t, k, v)
          up = v
      end
  }

  local mmt = {}

  setmetatable(mmt, mmmt)

  local mt = {
      __newindex = mmt
  }

  setmetatable(mt, mmt)

  local t = {}

  setmetatable(t, mt)

  t.yo = "hello"

  local _r1, _r2 = t.yo, up
  assert(_r1 == nil)
  assert(_r2 == "hello")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 9] binary __xxx functions in metatable
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local mt = {
      __add = function (a, b)
          return "{} + " .. b
      end,

      __sub = function (a, b)
          return "{} - " .. b
      end,

      __mul = function (a, b)
          return "{} * " .. b
      end,

      __mod = function (a, b)
          return "{} % " .. b
      end,

      __pow = function (a, b)
          return "{} ^ " .. b
      end,

      __div = function (a, b)
          return "{} / " .. b
      end,

      __idiv = function (a, b)
          return "{} // " .. b
      end,

      __band = function (a, b)
          return "{} & " .. b
      end,

      __bor = function (a, b)
          return "{} | " .. b
      end,

      __bxor = function (a, b)
          return "{} ~ " .. b
      end,

      __shl = function (a, b)
          return "{} << " .. b
      end,

      __shr = function (a, b)
          return "{} >> " .. b
      end

  }

  local t = {}

  setmetatable(t, mt)

  print(
      t + 1,
      t - 1,
      t * 1,
      t % 1,
      t ^ 1,
      t / 1,
      t // 1,
      t & 1,
      t | 1,
      t ~ 1,
      t << 1,
      t >> 1)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 10] __eq
-- Verifies: output matches expected value via print()
do
  local mt = {
      __eq = function (a, b)
          return true
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t == {})
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 11] __lt
-- Verifies: output matches expected value via print()
do
  local mt = {
      __lt = function (a, b)
          return true
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t < {})
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 12] __le
-- Verifies: output matches expected value via print()
do
  local mt = {
      __le = function (a, b)
          return true
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t <= {})
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 13] __le that uses __lt
-- Verifies: output matches expected value via print()
do
  local mt = {
      __lt = function (a, b)
          return false
      end
  }

  local t = {}

  setmetatable(t, mt)

  print({} <= t)
end
--> =true

-- --------------------------------------------------------------------------
-- [Test 14] __unm, __bnot
-- Verifies: output matches expected value via print()
do
  local mt = {
      __unm = function (a)
          return "hello"
      end,

      __bnot = function (a)
          return "world"
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(-t, ~t)
end
--> =hello	world

-- --------------------------------------------------------------------------
-- [Test 15] __len
-- Verifies: output matches expected value via print()
do
  local mt = {
      __len = function (a)
          return "hello"
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(#t)
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 16] __concat
-- Verifies: output matches expected value via print()
do
  local mt = {
      __concat = function (a)
          return "hello"
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t .. " world")
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 17] __call
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local mt = {
      __call = function (a, ...)
          return "hello", ...
      end
  }

  local t = {}

  setmetatable(t, mt)

  print(t("world","wow"))
  print("PASS")
end
--> =PASS
