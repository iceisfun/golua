-- Runtime error lines for expressions whose operator/field is split across
-- source lines. Reference Lua attributes the faulting instruction to the line
-- of the token that "owns" it (the right operand of a comparison, the last
-- '..' operator of a concat chain, the field of an indexed assignment target),
-- not the line where the statement/expression began. golua's AST compiler used
-- the expression's start line for these and reported the wrong line.
--
-- Each case computes the expected line relative to a debug.getinfo anchor so it
-- is independent of this file's layout. The printed delta must match reference.
local debug = require "debug"

-- Comparison: the OP_LT/EQ/LE faults; reference uses the right-operand line.
do
  local mt = {__lt = function() error("E", 2) end}
  local a, b = setmetatable({}, mt), setmetatable({}, mt)
  local base
  local _, err = pcall(function()
    base = debug.getinfo(1, "l").currentline
    return a <
      b
  end)
  print("cmp", tonumber(err:match(":(%d+):")) - base)
  --> =cmp	2
end

-- Concat chain: the folded OP_CONCAT faults; reference uses the last '..' line.
do
  local x = nil
  local base
  local _, err = pcall(function()
    base = debug.getinfo(1, "l").currentline
    return "a" ..
      "b" ..
      x
  end)
  print("cat", tonumber(err:match(":(%d+):")) - base)
  --> =cat	2
end

-- Indexed-assignment target: the store faults; reference uses the field line.
do
  local base
  local _, err = pcall(function()
    local t = {}
    base = debug.getinfo(1, "l").currentline
    t.a
      .b = 5
  end)
  print("asgn", tonumber(err:match(":(%d+):")) - base)
  --> =asgn	2
end
