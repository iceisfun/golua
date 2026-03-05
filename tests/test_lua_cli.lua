-- ==========================================================================
-- Fengari test extraction: Lua CLI interface
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: lua_cli
-- Total tests: 1
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] __newindex leaves nils
-- Verifies: all assert() calls pass without error
do
  local x = setmetatable({}, {
    __newindex = function(t,k,v)
      rawset(t,'_'..k,v)
    end
  })
  x.test = 4
  for k,v in pairs(x) do
    assert(k ~= "test", "found phantom key")
  end
  print("PASS")
end
--> =PASS
