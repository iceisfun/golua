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
  print(table.concat({1, 2, 3, 4, 5, 6, 7}, ",", 3, 5))
end
--> =3,4,5

-- --------------------------------------------------------------------------
-- [Test 2] table.pack
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print(table.pack(1, 2, 3))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 3] table.unpack
-- Verifies: output matches expected value via print()
do
  print(table.unpack({1, 2, 3, 4, 5}, 2, 4))
end
--> =2	3	4

-- --------------------------------------------------------------------------
-- [Test 4] table.insert
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 3, 4}
  table.insert(t, 5)
  table.insert(t, 2, 2)
  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 5] table.remove
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {1, 2, 3, 3, 4, 4}
  table.remove(t)
  table.remove(t, 3)
  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 6] table.move
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t1 = {3, 4, 5}
  local t2 = {1, 2, nil, nil, nil, 6}
  print(table.move(t1, 1, #t1, 3, t2))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 7] table.sort (<)
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {3, 1, 5, ['just'] = 'tofuckitup', 2, 4}
  table.sort(t)
  print(t)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 8] table.sort with cmp function
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local t = {3, 1, 5, ['just'] = 'tofuckitup', 2, 4}
  table.sort(t, function (a, b)
      return a > b
  end)
  print(t)
  print("PASS")
end
--> =PASS
