-- broken_fuzz_pcall_in_close_from_coroutine_close:
-- pcall inside __close is broken when invoked from coroutine.close
--
-- BROKEN: When a coroutine with a `<close>` local is closed via
-- `coroutine.close(co)`, any `pcall` inside the `__close` handler does NOT
-- catch errors raised inside its protected function — the error escapes
-- pcall entirely, aborting the handler and propagating as the return value
-- of `coroutine.close`. The scope-exit `<close>` path (without coroutine.close)
-- handles pcall correctly, so the defect is specific to the close-from-resume
-- path. Reproduces identically against lua5.4.8 reference, so it is a
-- golua bug, not a 5.4→5.5 gap.
--
-- Reference (lua5.5.0):
--   prints "caught: false"
--   coroutine.close(co) returns true
--
-- golua today:
--   no print from the __close handler
--   coroutine.close(co) returns false, "e"
--
-- Discovered: differential fuzz 2026-04-23 (coroutines_2). Top-priority bug
-- from this round — silently breaks error containment in close handlers.

local co = coroutine.create(function()
  local x <close> = setmetatable({}, {__close = function()
    local ok, err = pcall(function() error("e") end)
    -- pcall must return false with the error value; it must NOT propagate.
    assert(ok == false, "pcall should return false, got " .. tostring(ok))
    print("caught:", ok)
  end})
  coroutine.yield()
end)

coroutine.resume(co)
local closed, err = coroutine.close(co)
assert(closed == true, "coroutine.close should return true, got " ..
       tostring(closed) .. ", " .. tostring(err))
