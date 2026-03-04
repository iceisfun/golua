-- Test: db.lua - Hook presence in debug info
-- From: db.lua
-- What: Tests that hooks are identified as "hook" in debug.getinfo namewhat

do
  local count = 0
  local function f ()
    assert(debug.getinfo(1).namewhat == "hook")
    local sndline = string.match(debug.traceback(), "\n(.-)\n")
    assert(string.find(sndline, "hook"))
    count = count + 1
  end
  debug.sethook(f, "l")
  local a = 0
  _ENV.a = a
  a = 1
  debug.sethook()
  assert(count == 4)
end
