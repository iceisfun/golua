-- Test that binary operator line numbers match Lua 5.4's behavior.
-- In Lua 5.4, the left operand's discharge instruction gets the operator's
-- line, not the operand's line. The store instruction (SETTABUP) gets the
-- end line of the RHS expression.
local debug = require "debug"

local function getlines(s)
  local lines = {}
  local function f(event, line)
    if event == 'line' then lines[#lines+1] = line end
  end
  debug.sethook(f, "l"); load(s)(); debug.sethook()
  return lines
end

-- b[1] + b[1] across lines: GETI for left b[1] should get operator's line
local s = "     local b = {10}\n     a = b[1] \n + \n b[1]\n     b = 4\n  "
local lines = getlines(s)
-- Expected: {1, 3, 4, 3, 4, 5}
assert(#lines == 6, "expected 6 line events, got " .. #lines)
assert(lines[1] == 1)
assert(lines[2] == 3)  -- GETI left discharged at operator line
assert(lines[3] == 4)  -- GETI right on its own line
assert(lines[4] == 3)  -- ADD at operator line
assert(lines[5] == 4)  -- SETTABUP at expression end line
assert(lines[6] == 5)  -- LOADI for b = 4
print("PASS") --> PASS

-- Regression: the ADDI/SUBI optimization (target = local +/- smallint) must not
-- relabel the PREVIOUS statement's instruction line. operandToReg emits nothing
-- when the left operand is a local already in a register, so an unconditional
-- fixDischargedLine wrongly moved the prior statement's line forward — making an
-- error raised on that prior statement report the +/- statement's line instead.
do
  local t = setmetatable({}, {__index = function() error("boom", 2) end})
  local n = 0
  local expected
  local ok, err = pcall(function()
    expected = debug.getinfo(1, "l").currentline + 1
    local x = t.foo   -- faulting line; immediately followed by an ADDI statement
    n = n + 1
    return x
  end)
  print(ok, tonumber(err:match(":(%d+):")) == expected)
  --> =false	true
end

do
  local t = setmetatable({}, {__index = function() error("boom", 2) end})
  local n = 0
  local expected
  local ok, err = pcall(function()
    expected = debug.getinfo(1, "l").currentline + 1
    local x = t.foo   -- faulting line; immediately followed by a SUBI statement
    n = n - 1
    return x
  end)
  print(ok, tonumber(err:match(":(%d+):")) == expected)
  --> =false	true
end
