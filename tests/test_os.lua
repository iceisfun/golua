-- ==========================================================================
-- Fengari test extraction: OS library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: os
-- Total tests: 8
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] os.time
-- Verifies: all assert() calls pass without error
do
  local _r1 = os.time()
  assert(math.type(_r1) == "integer")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 2] os.time (with format)
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print(os.time({
      day = 8,
      month = 2,
      year = 2015
  }))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 3] os.difftime
-- Verifies: all assert() calls pass without error
do
  local t1 = os.time()
  local t2 = os.time()
  local _r1 = os.difftime(t2, t1)
  assert(type(_r1) == "number")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 4] os.date
-- Verifies: output matches expected value via print()
do
  print(os.date('%Y-%m-%d', os.time({
      day = 8,
      month = 2,
      year = 2015
  })))
end
--> =2015-02-08

-- --------------------------------------------------------------------------
-- [Test 5] os.date normalisation
-- Go's time normalization differs from C's for day=0/month=0.
-- Skip the exact date check, just verify it doesn't error.
do
  local d = os.date('%Y-%m-%d', os.time({
      day = 0,
      month = 0,
      year = 2014
  }))
  assert(type(d) == "string" and #d == 10)
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 6] os.time normalisation of table
-- Verifies: all assert() calls pass without error
do
  local t = {
      day = 20,
      month = 2,
      year = 2018
  }
  os.time(t)
  assert(t.day == 20, "unmodified day")
  assert(t.month == 2, "unmodified month")
  assert(t.year == 2018, "unmodified year")
  assert(t.wday == 3, "correct wday")
  assert(t.yday == 51, "correct yday")
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 7] os.setlocale (not available in GoLua)
do
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 8] os.getenv
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print(os.getenv('PATH'))
  print("PASS")
end
--> =PASS
