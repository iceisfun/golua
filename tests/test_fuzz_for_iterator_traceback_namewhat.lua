-- test_fuzz_for_iterator_traceback_namewhat:
-- A generic-for iterator frame in a stack traceback must be labelled
-- "in for iterator 'for iterator'", not "in function 'for iterator'".
-- Lua 5.5's funcnamefromcode returns namewhat "for iterator" for the
-- OP_TFORCALL frame; the traceback formatter must honor that namewhat.
--
-- Discovered: differential scout 2026-05-20 (control-flow agent).

local function it() error("boom") end
local tb = select(2, xpcall(function()
  for _ in it do end
end, debug.traceback))

assert(tb:find("in for iterator 'for iterator'", 1, true),
  "iterator frame must be 'in for iterator', got:\n" .. tb)
assert(not tb:find("in function 'for iterator'", 1, true),
  "iterator frame must not be 'in function', got:\n" .. tb)

print("ok")
