-- "too many local variables" compile error should include line number,
-- "in main function", and "near" token (matching Lua 5.4 format)
local code = "local "
for i = 1, 201 do
  code = code .. "a" .. i
  if i < 201 then code = code .. ", " end
end
local f, msg = load(code)
assert(f == nil)
-- Check line number present
assert(msg:find(":1:"), "expected line number :1: in error, got: " .. msg)
-- Check "in main function" present
assert(msg:find("in main function"), "expected 'in main function' in error, got: " .. msg)
print("OK")
