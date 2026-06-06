-- broken_fuzz_coroutine_close_self: coroutine.close(self) should succeed in 5.5
--
-- BROKEN: Lua 5.5 allows a coroutine to close itself — the call terminates
-- the coroutine cleanly and the outer resume returns `true`. golua still
-- errors with "cannot close a running coroutine" (5.4 semantics).
--
-- Reference (lua5.5.0):
--   co = create(function() coroutine.close(co) end)
--   resume(co) → true
--   status(co) → "dead"
--
-- golua today:
--   resume(co) → false, "...cannot close a running coroutine"
--
-- Already noted as a known gap in project_v2_release.md; this test file
-- codifies the expected 5.5 behavior.
--
-- Discovered: differential fuzz 2026-04-23 (coroutines_1).

local co
co = coroutine.create(function()
  coroutine.close(co)
  error("must not reach this line")
end)

local ok = coroutine.resume(co)
assert(ok == true, "resume should return true, got " .. tostring(ok))
assert(coroutine.status(co) == "dead",
       "status should be 'dead', got " .. tostring(coroutine.status(co)))
