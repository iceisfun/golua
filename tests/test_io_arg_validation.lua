-- Test io.open argument validation and coercion

-- Fix 1: io.open should reject non-string/non-number types
local ok, err = pcall(io.open, true)
assert(not ok, "io.open(true) should error, got: " .. tostring(ok))
assert(string.find(err, "string expected, got boolean"), "wrong error: " .. tostring(err))

local ok2, err2 = pcall(io.open, {})
assert(not ok2, "io.open({}) should error")
assert(string.find(err2, "string expected, got table"), "wrong error: " .. tostring(err2))

-- io.open should coerce numbers to strings (tries to open file named "12345")
-- This will fail to open but should not error on argument type
local ok3, err3 = pcall(io.open, 12345)
assert(ok3, "io.open(12345) should not throw, got: " .. tostring(err3))
-- ok3 is true because pcall succeeded; the return is nil + error message (file not found)

-- Fix 2: io.input/io.output should coerce numbers to filenames
-- io.input(number) should try to open a file with that name (and fail with file error, not type error)
local ok4, err4 = pcall(io.input, 42)
assert(not ok4, "io.input(42) should error (file not found)")
assert(string.find(err4, "cannot open file '42'"), "wrong error for io.input(42): " .. tostring(err4))

local ok5, err5 = pcall(io.output, 99)
-- io.output tries to open for writing, so it may succeed or fail depending on permissions
-- but it should NOT error with "FILE* expected" - it should try to open file "99"
-- We just check it doesn't give a type error
if not ok5 then
  assert(not string.find(err5, "FILE%* expected"), "io.output(99) should coerce number, got: " .. tostring(err5))
end
-- Clean up any file created by io.output(99)
os.remove("99")

print("PASS")
