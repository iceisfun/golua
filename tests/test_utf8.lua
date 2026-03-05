-- ==========================================================================
-- Fengari test extraction: UTF-8 library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: utf8
-- Total tests: 5
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] utf8.offset
-- Verifies: output matches expected value via print()
do
  print(utf8.offset("( ͡° ͜ʖ ͡° )", 5))
end
--> =7

-- --------------------------------------------------------------------------
-- [Test 2] utf8.codepoint
-- Verifies: output matches expected value via print()
do
  print(utf8.codepoint("( ͡° ͜ʖ ͡° )", 5, 8))
end
--> =176	32	860

-- --------------------------------------------------------------------------
-- [Test 3] utf8.char
-- Verifies: output matches expected value via print()
do
  print(utf8.char(40, 32, 865, 176, 32, 860, 662, 32, 865, 176, 32, 41))
end
--> =( ͡° ͜ʖ ͡° )

-- --------------------------------------------------------------------------
-- [Test 4] utf8.len
-- Verifies: output matches expected value via print()
do
  print(utf8.len("( ͡° ͜ʖ ͡° )"))
end
--> =12

-- --------------------------------------------------------------------------
-- [Test 5] utf8.codes
-- Verifies: output matches expected value via print()
do
  local s = "( ͡° ͜ʖ ͡° )"
  local results = ""
  for p, c in utf8.codes(s) do
      results = results .. "[" .. p .. "," .. c .. "] "
  end
  print(results)
end
--> =[1,40] [2,32] [3,865] [5,176] [7,32] [8,860] [10,662] [12,32] [13,865] [15,176] [17,32] [18,41] 
