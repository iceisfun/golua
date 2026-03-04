-- Test: files.lua - Yielding during dofile
-- From: files.lua
-- What: Tests that coroutine.yield works inside a file loaded with dofile

do
  local file = os.tmpname()

  local f = assert(io.open(file, "w"))
  f:write[[
local x, z = coroutine.yield(10)
local y = coroutine.yield(20)
return x + y * z
]]
  assert(f:close())
  f = coroutine.wrap(dofile)
  assert(f(file) == 10)
  assert(f(100, 101) == 20)
  assert(f(200) == 100 + 200 * 101)

  os.remove(file)
end
