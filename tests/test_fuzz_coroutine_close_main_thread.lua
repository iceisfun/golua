-- Lua 5.5:
--   coroutine.close(mainthread) -> error "cannot close main thread"
--   coroutine.close()           -> defaults to current thread (main here)
--                                  -> same error
-- golua previously reported "cannot close a running coroutine" and rejected
-- the no-arg form with "thread expected, got no value".

local main = coroutine.running()

local ok, err = pcall(coroutine.close, main)
assert(not ok, "closing main should fail")
assert(tostring(err):find("cannot close main thread") ~= nil,
  "expected 'cannot close main thread', got: " .. tostring(err))

local ok2, err2 = pcall(coroutine.close)
assert(not ok2, "no-arg close should fail when running on main")
assert(tostring(err2):find("cannot close main thread") ~= nil,
  "expected 'cannot close main thread' for no-arg, got: " .. tostring(err2))

print("PASSED")
