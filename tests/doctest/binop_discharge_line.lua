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
