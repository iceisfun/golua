-- Test that debug.traceback("") includes leading newline
-- Lua 5.4: debug.traceback("") returns "\nstack traceback:\n\t..."
-- The empty string message should still produce a leading newline.

local tb = debug.traceback("")
assert(tb:sub(1, 1) == "\n", "traceback('') should start with newline, got: " .. string.format("%q", tb:sub(1,20)))

-- Non-empty message should also have newline before "stack traceback:"
local tb2 = debug.traceback("hello")
assert(tb2:sub(1, 5) == "hello", "traceback('hello') should start with 'hello'")
assert(tb2:find("hello\nstack traceback:") == 1, "traceback('hello') missing newline before stack traceback")

-- nil message should NOT have leading newline
local tb3 = debug.traceback()
assert(tb3:sub(1, 5) == "stack", "traceback() should start with 'stack', got: " .. string.format("%q", tb3:sub(1,20)))

print("PASS")
