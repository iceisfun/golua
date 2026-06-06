-- test_fuzz_getlocal_for_slots: numeric-for internal slot layout (Lua 5.5)
--
-- Lua 5.5 represents numeric-for state with 2 control slots and folds the
-- visible loop variable into the control register, so inside the body the
-- slots are: (for state), (for state), i, <body locals>...
--
-- Reference (lua5.5.0):  (for state), (for state), i, names, ...
--
-- This regressed when golua still emitted the 3-slot Lua 5.4 layout; the fix
-- reworked OP_FORPREP/OP_FORLOOP slot allocation in compiler/compile_control.go
-- and vm/vm_exec.go. Verified identical to lua5.5.0.
--
-- Discovered: differential fuzz 2026-04-23 (debuglib_3); fixed 2026-06-06.

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
    assert(names[4] == "names",
           "slot 4 should be the body local 'names', got: " .. tostring(names[4]))
  end
  print("OK")
end
probe()
