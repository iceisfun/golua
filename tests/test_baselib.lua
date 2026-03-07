-- ==========================================================================
-- Fengari test extraction: Base library functions (print, type, pcall, etc.)
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: baselib
-- Total tests: 17
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.

-- [Test 1] print
-- Verifies print doesn't error (output goes to stdout)
do
  print("hello", "world", 123)
end

-- --------------------------------------------------------------------------
-- [Test 2] setmetatable, getmetatable
do
  local mt = {
      __index = function ()
          return "hello"
      end
  }

  local t = {}
  setmetatable(t, mt);

  assert(t[1] == "hello")
  assert(type(getmetatable(t)) == "table")
end

-- --------------------------------------------------------------------------
-- [Test 3] rawequal
-- rawequal bypasses __eq metamethod
do
  local mt = {
      __eq = function ()
          return true
      end
  }

  local t1 = {}
  local t2 = {}
  setmetatable(t1, mt);

  assert(rawequal(t1, t2) == false)
  assert(t1 == t2)
end

-- --------------------------------------------------------------------------
-- [Test 4] rawset, rawget
-- rawset/rawget bypass __newindex/__index metamethods
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

  assert(rawget(t, "yo") == "hello" and t["yo"] == "hello")
  assert(rawget(t, "yoyo") == "bye" and t["yoyo"] == "bye")
end

-- --------------------------------------------------------------------------
-- [Test 5] type
do
  assert(type(1) == "number")
  assert(type(true) == "boolean")
  assert(type("hello") == "string")
  assert(type({}) == "table")
  assert(type(nil) == "nil")
end

-- --------------------------------------------------------------------------
-- [Test 6] error
do
  local ok, msg = pcall(error, "you fucked up")
  assert(not ok and string.find(msg, "you fucked up"))
end

-- --------------------------------------------------------------------------
-- [Test 7] error, protected
do
  local ok, msg = pcall(error, "you fucked up")
  assert(not ok and string.find(msg, "you fucked up"))
end

-- --------------------------------------------------------------------------
-- [Test 8] pcall
do
  local willFail = function ()
      error("you fucked up")
  end

  local ok, msg = pcall(willFail)
  assert(not ok and string.find(msg, "you fucked up"))
end

-- --------------------------------------------------------------------------
-- [Test 9] xpcall
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
end

-- --------------------------------------------------------------------------
-- [Test 10] ipairs
do
  local t = {1, 2, 3, 4, 5, ['yo'] = 'lo'}

  local sum = 0
  for i, v in ipairs(t) do
      sum = sum + v
  end

  assert(sum == 15)
end

-- --------------------------------------------------------------------------
-- [Test 11] select
do
  assert(select('#', 1, 2, 3) == 3)
  local a, b = select(2, 1, 2, 3)
  assert(a == 2 and b == 3)
  local c, d = select(-2, 1, 2, 3)
  assert(c == 2 and d == 3)
end

-- --------------------------------------------------------------------------
-- [Test 12] tonumber
do
  assert(tonumber('foo') == nil)
  assert(tonumber('123') == 123)
  assert(tonumber('12.3') == 12.3)
  assert(tonumber('az', 36) == 395)
  assert(tonumber('10', 2) == 2)
end

-- --------------------------------------------------------------------------
-- [Test 13] assert (pcall catches assert failure)
do
  local ok, msg = pcall(assert, 1 < 0, "this doesn't makes sense")
  assert(not ok and string.find(msg, "this doesn't makes sense"))
end

-- --------------------------------------------------------------------------
-- [Test 14] rawlen
do
  assert(rawlen({1, 2, 3}) == 3)
  assert(rawlen('hello') == 5)
end

-- --------------------------------------------------------------------------
-- [Test 15] next
do
  local total = 0
  local t = { 1, two = 2, 3, four = 4 }

  for k,v in next, t, nil do
      total = total + v
  end

  assert(total == 10)
end

-- --------------------------------------------------------------------------
-- [Test 16] pairs
do
  local total = 0
  local t = { 1, two = 2, 3, four = 4 }

  for k,v in pairs(t) do
      total = total + v
  end

  assert(total == 10)
end

-- --------------------------------------------------------------------------
-- [Test 17] pairs with __pairs
-- __pairs metamethod overrides the default iterator
do
  local total = 0

  local mt = {
      __pairs = function(t)
          return next, {5, 6, 7, 8}, nil
      end
  }

  local t = { 1, two = 2, 3, four = 4 }
  setmetatable(t, mt)

  for k,v in pairs(t) do
      total = total + v
  end

  assert(total == 26)
end
