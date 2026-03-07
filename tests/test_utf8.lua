-- ==========================================================================
-- Fengari test extraction: UTF-8 library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: utf8
-- Total tests: 5
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.

-- [Test 1] utf8.offset
do
  assert(utf8.offset("( ͡° ͜ʖ ͡° )", 5) == 7)
end

-- --------------------------------------------------------------------------
-- [Test 2] utf8.codepoint (multi-return)
do
  local a, b, c = utf8.codepoint("( ͡° ͜ʖ ͡° )", 5, 8)
  assert(a == 176 and b == 32 and c == 860)
end

-- --------------------------------------------------------------------------
-- [Test 3] utf8.char
do
  assert(utf8.char(40, 32, 865, 176, 32, 860, 662, 32, 865, 176, 32, 41) == "( ͡° ͜ʖ ͡° )")
end

-- --------------------------------------------------------------------------
-- [Test 4] utf8.len
do
  assert(utf8.len("( ͡° ͜ʖ ͡° )") == 12)
end

-- --------------------------------------------------------------------------
-- [Test 5] utf8.codes
do
  local s = "( ͡° ͜ʖ ͡° )"
  local results = {}
  for p, c in utf8.codes(s) do
      results[#results + 1] = "[" .. p .. "," .. c .. "]"
  end
  assert(table.concat(results, " ") == "[1,40] [2,32] [3,865] [5,176] [7,32] [8,860] [10,662] [12,32] [13,865] [15,176] [17,32] [18,41]")
end
