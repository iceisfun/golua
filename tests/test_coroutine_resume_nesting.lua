-- Test: cstack.lua - Nesting of resuming yielded coroutines
-- From: cstack.lua
-- What: Tests stack overflow from deeply nested coroutine resume chains

do
  local count = 0
  local function body ()
    coroutine.yield()
    local f = coroutine.wrap(body)
    f();
    count = count + 1
    f()
  end
  local f = coroutine.wrap(body)
  f()
  assert(not pcall(f))
end
