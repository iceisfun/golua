-- ==========================================================================
-- Fengari test extraction: Metamethod dispatch (__index, __add, etc.)
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: metamethods
-- Total tests: 17
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.

-- [Test 1] __index, __newindex: with actual table
do
  local t = {yo=1}
  assert(t.yo == 1)
  assert(t.lo == nil)
end

-- --------------------------------------------------------------------------
-- [Test 2] __newindex: with non table
-- Indexing a string for assignment should raise an error
do
  local ok, err = pcall(function()
    local t = "a string"
    t.yo = "hello"
  end)
  assert(not ok, "expected error when indexing string for newindex")
end

-- --------------------------------------------------------------------------
-- [Test 3] __index function in metatable
do
  local mt = {
      __index = function (table, key)
          return "__index"
      end
  }

  local t = {}
  setmetatable(t, mt)
  assert(t.yo == "__index")
end

-- --------------------------------------------------------------------------
-- [Test 4] __newindex function in metatable
-- __newindex intercepts assignment, so t.yo remains nil
do
  local mt = {
      __newindex = function (table, key, value)
          return "__newindex"
      end
  }

  local t = {}
  setmetatable(t, mt)
  t.yo = "hello"
  assert(t.yo == nil)
end

-- --------------------------------------------------------------------------
-- [Test 5] __index table in metatable
do
  local mmt = { yo = "hello" }
  local mt = { __index = mmt }
  local t = {}
  setmetatable(t, mt)
  assert(t.yo == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 6] __newindex table in metatable
-- Assignment goes to the __newindex table, not t
do
  local mmt = { yo = "hello" }
  local mt = { __newindex = mmt }
  local t = {}
  setmetatable(t, mt)
  t.yo = "world"
  assert(t.yo == nil)
  assert(mmt.yo == "world")
end

-- --------------------------------------------------------------------------
-- [Test 7] __index table with own metatable
-- Chain: t -> mt.__index=mmt -> mmmt.__index(function)
do
  local mmmt = {
      __index = function (t, k)
          return "hello"
      end
  }

  local mmt = { yoo = "bye" }
  setmetatable(mmt, mmmt)

  local mt = { __index = mmt }
  local t = {}
  setmetatable(t, mt)
  assert(t.yo == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 8] __newindex table with own metatable
-- Chain: t -> mt.__newindex=mmt -> mmmt.__newindex(function)
do
  local up = nil

  local mmmt = {
      __newindex = function (t, k, v)
          up = v
      end
  }

  local mmt = {}
  setmetatable(mmt, mmmt)

  local mt = { __newindex = mmt }
  setmetatable(mt, mmt)

  local t = {}
  setmetatable(t, mt)
  t.yo = "hello"
  assert(t.yo == nil)
  assert(up == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 9] binary __xxx functions in metatable
do
  local mt = {
      __add = function (a, b) return "{} + " .. b end,
      __sub = function (a, b) return "{} - " .. b end,
      __mul = function (a, b) return "{} * " .. b end,
      __mod = function (a, b) return "{} % " .. b end,
      __pow = function (a, b) return "{} ^ " .. b end,
      __div = function (a, b) return "{} / " .. b end,
      __idiv = function (a, b) return "{} // " .. b end,
      __band = function (a, b) return "{} & " .. b end,
      __bor = function (a, b) return "{} | " .. b end,
      __bxor = function (a, b) return "{} ~ " .. b end,
      __shl = function (a, b) return "{} << " .. b end,
      __shr = function (a, b) return "{} >> " .. b end
  }

  local t = {}
  setmetatable(t, mt)

  assert(t + 1 == "{} + 1")
  assert(t - 1 == "{} - 1")
  assert(t * 1 == "{} * 1")
  assert(t % 1 == "{} % 1")
  assert(t ^ 1 == "{} ^ 1")
  assert(t / 1 == "{} / 1")
  assert(t // 1 == "{} // 1")
  assert(t & 1 == "{} & 1")
  assert(t | 1 == "{} | 1")
  assert(t ~ 1 == "{} ~ 1")
  assert(t << 1 == "{} << 1")
  assert(t >> 1 == "{} >> 1")
end

-- --------------------------------------------------------------------------
-- [Test 10] __eq
do
  local mt = { __eq = function (a, b) return true end }
  local t = {}
  setmetatable(t, mt)
  assert(t == {})
end

-- --------------------------------------------------------------------------
-- [Test 11] __lt
do
  local mt = { __lt = function (a, b) return true end }
  local t = {}
  setmetatable(t, mt)
  assert(t < {})
end

-- --------------------------------------------------------------------------
-- [Test 12] __le
do
  local mt = { __le = function (a, b) return true end }
  local t = {}
  setmetatable(t, mt)
  assert(t <= {})
end

-- --------------------------------------------------------------------------
-- [Test 13] __le that uses __lt
-- In Lua 5.4, a <= b tries __le first, then falls back to not (b < a)
do
  local mt = { __lt = function (a, b) return false end }
  local t = {}
  setmetatable(t, mt)
  assert({} <= t)
end

-- --------------------------------------------------------------------------
-- [Test 14] __unm, __bnot
do
  local mt = {
      __unm = function (a) return "hello" end,
      __bnot = function (a) return "world" end
  }

  local t = {}
  setmetatable(t, mt)
  assert(-t == "hello")
  assert(~t == "world")
end

-- --------------------------------------------------------------------------
-- [Test 15] __len
do
  local mt = { __len = function (a) return "hello" end }
  local t = {}
  setmetatable(t, mt)
  assert(#t == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 16] __concat
do
  local mt = { __concat = function (a) return "hello" end }
  local t = {}
  setmetatable(t, mt)
  assert(t .. " world" == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 17] __call
do
  local mt = {
      __call = function (a, ...)
          return "hello", ...
      end
  }

  local t = {}
  setmetatable(t, mt)
  local r1, r2, r3 = t("world","wow")
  assert(r1 == "hello" and r2 == "world" and r3 == "wow")
end
