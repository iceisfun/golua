-- Test: constructs.lua - Short-circuit optimization correctness
-- From: constructs.lua
-- What: Tests that short-circuit evaluation of and/or produces correct values for all combinations

do
  local basiccases = {
    {"nil", nil},
    {"false", false},
    {"true", true},
    {"10", 10},
    {"(0==_ENV.GLOB1)", 0 == _ENV.GLOB1},
  }
  -- Creates all combinations of (cases[i] op cases[n-i]) and tests them
  for _, c1 in ipairs(basiccases) do
    for _, c2 in ipairs(basiccases) do
      local code_and = string.format("return (%s) and (%s)", c1[1], c2[1])
      local code_or  = string.format("return (%s) or (%s)", c1[1], c2[1])
      local expected_and = c1[2] and c2[2]
      local expected_or  = c1[2] or c2[2]
      assert(load(code_and)() == expected_and,
             "failed: " .. code_and)
      assert(load(code_or)() == expected_or,
             "failed: " .. code_or)
    end
  end
end
