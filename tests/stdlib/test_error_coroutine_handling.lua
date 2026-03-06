-- Test: errors.lua - Coroutine error handling
-- From: errors.lua
-- What: Tests error propagation in coroutines and yield-across-C-boundary errors

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  local function f (n)
    local c = coroutine.create(f)
    local a,b = coroutine.resume(c)
    return b
  end
  assert(string.find(f(), "stack overflow"))
  checkmessage("coroutine.yield()", "outside a coroutine")
end
