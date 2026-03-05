-- warn() should be disabled by default in Lua 5.4.
-- Lua 5.4 Reference: The standard library provides a default warn function
-- (warnfoff) that starts with warnings OFF. "@on" enables them.

-- Test 1: warn should not produce output when disabled (default)
-- We can't easily capture warn output, but we can test the control messages
-- and verify warn doesn't crash when called while disabled.
warn("this should not appear") -- should be silently ignored

-- Test 2: @on enables warnings
warn("@on")
-- After @on, warnings are active (we just verify it doesn't error)
warn("@off") -- disable again

-- Test 3: verify @off works
warn("@off")
warn("this should also not appear")

print("PASS")
