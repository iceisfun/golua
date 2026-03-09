-- Test: debug.traceback with negative level should not crash
local result = debug.traceback("msg", -1)
assert(type(result) == "string", "expected string result")
assert(result:find("msg"), "expected message in result")
assert(result:find("stack traceback:"), "expected stack traceback header")

-- Also test with very negative level
local result2 = debug.traceback("hello", -100)
assert(type(result2) == "string")

-- Test without message
local result3 = debug.traceback(nil, -1)
assert(type(result3) == "string")

print("OK")
