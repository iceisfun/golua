-- ==========================================================================
-- Fengari test extraction: Table library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: table
-- Total tests: 8
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] table.concat
-- Verifies: output matches expected value via print()
do
  assert(table.concat({1, 2, 3, 4, 5, 6, 7}, ",", 3, 5) == "3,4,5")
end

-- --------------------------------------------------------------------------
-- [Test 2] table.pack
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local p = table.pack(1, 2, 3)
  assert(p.n == 3 and p[1] == 1 and p[2] == 2 and p[3] == 3)
end

-- --------------------------------------------------------------------------
-- [Test 3] table.unpack
-- Verifies: output matches expected value via print()
do
  local a, b, c = table.unpack({1, 2, 3, 4, 5}, 2, 4)
  assert(a == 2 and b == 3 and c == 4)
end

-- --------------------------------------------------------------------------
-- [Test 4] table.insert
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 3, 4}
  table.insert(t, 5)
  table.insert(t, 2, 2)
  assert(t[1] == 1 and t[2] == 2 and t[3] == 3 and t[4] == 4 and t[5] == 5)
end

-- --------------------------------------------------------------------------
-- [Test 5] table.remove
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 3, 4, 4}
  table.remove(t)
  table.remove(t, 3)
  assert(#t == 4 and t[1] == 1 and t[2] == 2 and t[3] == 3 and t[4] == 4)
end

-- --------------------------------------------------------------------------
-- [Test 6] table.move
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t1 = {3, 4, 5}
  local t2 = {1, 2, nil, nil, nil, 6}
  local r = table.move(t1, 1, #t1, 3, t2)
  assert(r == t2)
  assert(t2[1] == 1 and t2[2] == 2 and t2[3] == 3 and t2[4] == 4 and t2[5] == 5 and t2[6] == 6)
end

-- --------------------------------------------------------------------------
-- [Test 7] table.sort (<)
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {3, 1, 5, ['just'] = 'tofuckitup', 2, 4}
  table.sort(t)
  assert(t[1] == 1 and t[2] == 2 and t[3] == 3 and t[4] == 4 and t[5] == 5)
end

-- --------------------------------------------------------------------------
-- [Test 8] table.sort with cmp function
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {3, 1, 5, ['just'] = 'tofuckitup', 2, 4}
  table.sort(t, function (a, b)
      return a > b
  end)
  assert(t[1] == 5 and t[2] == 4 and t[3] == 3 and t[4] == 2 and t[5] == 1)
end
