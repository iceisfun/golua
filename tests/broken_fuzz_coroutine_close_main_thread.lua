-- broken_fuzz_coroutine_close_main_thread:
-- coroutine.close(main_thread) and coroutine.close() with no args
--
-- BROKEN: Two Lua 5.5-introduced behaviors missing in golua:
-- 1. Closing the main thread must error with "cannot close main thread"
--    (5.5-specific message). golua falls through to the generic
--    "cannot close a running coroutine" path.
-- 2. `coroutine.close()` with no argument should default to the current
--    thread and error the same way. golua's arg-checker rejects with
--    "thread expected, got no value" (5.4 behavior).
--
-- Reference (lua5.5.0):
--   pcall(coroutine.close, coroutine.running())
--     → false, "...cannot close main thread"
--   pcall(coroutine.close)
--     → false, "...cannot close main thread"
--
-- golua today:
--   pcall(coroutine.close, coroutine.running())
--     → false, "...cannot close a running coroutine"
--   pcall(coroutine.close)
--     → false, "...bad argument #1 to 'coroutine.close' (thread expected, got no value)"
--
-- Fix in stdlib/coroutine.go: special-case the main thread in close(), and
-- treat missing arg as "current thread" before the type check.
--
-- Discovered: differential fuzz 2026-04-23 (coroutines_4).

local main = coroutine.running()
local ok1, err1 = pcall(coroutine.close, main)
assert(not ok1)
assert(tostring(err1):find("cannot close main thread"),
       "expected 'cannot close main thread', got: " .. tostring(err1))

local ok2, err2 = pcall(coroutine.close)
assert(not ok2)
assert(tostring(err2):find("cannot close main thread"),
       "expected 'cannot close main thread' for no-arg form, got: " ..
       tostring(err2))
