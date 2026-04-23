-- broken_fuzz_error_nil_via_resume:
-- error(nil) surfaces as raw nil through coroutine.resume (should be "<no error object>")
--
-- BROKEN: Lua 5.5's `error(nil)` (and other non-standard error values) must
-- surface as the literal string `"<no error object>"` in the error path.
-- golua correctly applies this when the error is captured via `pcall`
-- (commit 2b53f2d — feat: error(nil) returns "<no error object>"), but the
-- conversion is missing on the `coroutine.resume` return-value path — the
-- raw `nil` propagates unchanged.
--
-- Reference (lua5.5.0):
--   co = create(function() error(nil) end)
--   resume(co) → false, "<no error object>"   (string)
--
-- golua today:
--   resume(co) → false, nil                    (nil type)
--
-- Accounted for ~9 fuzz divergences in the coroutine pass, all same cause.
-- Fix should share the same value-conversion helper as the pcall path.
--
-- Discovered: differential fuzz 2026-04-23 (coroutines_3).

local co = coroutine.create(function() error(nil) end)
local ok, err = coroutine.resume(co)
assert(ok == false, "resume should return false, got " .. tostring(ok))
assert(type(err) == "string",
       "error should be string '<no error object>', got type " .. type(err) ..
       " value " .. tostring(err))
assert(err == "<no error object>",
       "error should be '<no error object>', got: " .. tostring(err))
