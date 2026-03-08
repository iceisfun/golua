-- Test exec.spawn with timed wait

local p = exec.spawn("sleep", "60")

-- Timed wait should return not-done quickly
local result, done = p:wait(100)  -- 100ms timeout
assert(done == false, "should not be done after 100ms")
assert(result == nil, "result should be nil when not done")

-- Kill and then wait should succeed
p:kill()
local result2 = p:wait()
assert(result2.success == false, "killed process should not succeed")

print("OK")
