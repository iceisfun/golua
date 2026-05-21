-- test_fuzz_comparison_arg_traceback_name:
-- A function call whose argument is a comparison expression (==, ~=, <, >,
-- <=, >=) must keep the callee's name in stack tracebacks. A comparison
-- argument compiles to a comparison + JMP between the callee load and the
-- CALL; the traceback name-resolution scan must skip the JMP (and other
-- non-register-writing opcodes) rather than mistake its offset bits for
-- the instruction that defined the callee.
--
-- (Lua 5.4 renders a C-function callee as "in function 'name'"; Lua 5.5
-- distinguishes "in global 'name'". This 5.4.8-branch test asserts the 5.4
-- wording.)
--
-- Discovered: differential scout 2026-05-20 (control-flow agent, s3#5).

-- global callee with a comparison argument keeps the name 'assert'
local tb = select(2, xpcall(function() assert(1 == 2, "x") end, debug.traceback))
assert(tb:find("[C]: in function 'assert'", 1, true),
  "comparison-arg call must keep the callee name, got:\n" .. tb)

-- works for every comparison operator
for _, p in ipairs{
  function() assert(1 ~= 1, "x") end,
  function() assert(2 > 3, "x") end,
  function() assert(2 < 1, "x") end,
  function() assert(2 <= 1, "x") end,
  function() assert(1 >= 2, "x") end,
} do
  local t = select(2, xpcall(p, debug.traceback))
  assert(t:find("[C]: in function 'assert'", 1, true),
    "every comparison operator must keep the name, got:\n" .. t)
end

-- a non-comparison argument still resolves (regression guard)
local tb2 = select(2, xpcall(function() assert(false, "x") end, debug.traceback))
assert(tb2:find("[C]: in function 'assert'", 1, true), "plain call lost its name:\n" .. tb2)

print("ok")
