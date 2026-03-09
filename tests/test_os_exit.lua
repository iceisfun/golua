-- Test os.exit

-- os.exit exists
assert(type(os.exit) == "function")

-- os.exit is not caught by pcall (it propagates through ProtectedCall)
-- The test harness catches LuaExitError as normal termination.

-- Test that os.exit(0) terminates execution.
-- Lines after os.exit should never be reached.
os.exit(0)
error("SHOULD NOT REACH HERE")
