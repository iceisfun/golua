-- debug.traceback should validate level argument type/index like Lua 5.4.

do
  local ok, err = pcall(debug.traceback, coroutine.running(), 0, "")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #3", 1, true) ~= nil, msg)
  assert(string.find(msg, "debug.traceback", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(debug.traceback, nil, coroutine.running(), "")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "debug.traceback", 1, true) ~= nil, msg)
end
