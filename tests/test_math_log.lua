-- test_math_log: math.log edge cases

assert(math.log(1) == 0.0, "log(1) should be 0.0")
assert(math.log(8, 2) == 3.0, "log(8,2) should be 3.0")

-- no args should error
assert(not pcall(math.log), "log() no args")

-- table arg should error
assert(not pcall(math.log, {}), "log({}) should error")

-- bad second arg should error and mention arg #2
local ok, err = pcall(math.log, 3, {})
assert(not ok, "log(3, {}) should error")
assert(string.find(err, "#2"), string.format("log error should mention arg #2, got: %s", err))
