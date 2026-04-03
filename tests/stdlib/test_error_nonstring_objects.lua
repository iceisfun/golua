-- Test: errors.lua - Non-string error objects
-- From: errors.lua
-- What: Tests error() with table and nil values, xpcall with error object transformation

do
  local t = {}
  local res, msg = pcall(function () error(t) end)
  assert(not res and msg == t)

  -- Lua 5.5: error(nil) is replaced by the string "<no error object>"
  res, msg = pcall(function () error(nil) end)
  assert(not res and msg == "<no error object>")

  local function f() error{msg='x'} end
  res, msg = xpcall(f, function (r) return {msg=r.msg..'y'} end)
  assert(msg.msg == 'xy')
end
