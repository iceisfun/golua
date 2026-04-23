-- broken_fuzz_getlocal_for_slots: numeric-for internal slot layout (5.4 vs 5.5)
--
-- BROKEN: Lua 5.5 represents numeric-for state with 2 control slots (a fused
-- init/limit/step arrangement), while golua still emits 3 control slots like
-- Lua 5.4. Additionally, 5.5 adds a `(vararg table)` slot at index 1 for
-- vararg functions, which golua does not produce. The net effect is that
-- `debug.getlocal` returns user-local indices shifted relative to the
-- reference — any tool that iterates `getlocal(1), getlocal(2), ...` sees
-- different names/values.
--
-- Reference (lua5.5.0):  (for state), (for state), i, (temporary), ...
-- golua today:           (for state), (for state), (for state), i, ...
--
-- Fix requires compiler bytecode layout change around OP_FORPREP/OP_FORLOOP
-- slot allocation. Non-trivial — likely needs coordinated changes in
-- compiler/, vm/vm_exec.go, and debug/getlocal.
--
-- Discovered: differential fuzz 2026-04-23 (debuglib_3).

local function probe()
  for i = 8, 10 do
    -- Names of slots at runtime inside the loop body.
    local names = {}
    for k = 1, 10 do
      local name = debug.getlocal(1, k)
      if not name then break end
      names[k] = name
    end
    -- In Lua 5.5 slot k=3 inside the loop body should be `i` (the user local),
    -- since only 2 control slots precede it.
    assert(names[3] == "i",
           "slot 3 should be user local 'i', got: " .. tostring(names[3]))
  end
end
probe()
