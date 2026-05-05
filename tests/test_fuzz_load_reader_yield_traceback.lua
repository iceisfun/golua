-- broken_fuzz_load_reader_yield_traceback:
-- load() returns its 2nd value (error message) with a stack traceback
-- appended when the reader function yields across a C-call boundary.
-- Reference Lua returns just the bare error message.
--
-- BROKEN: stdlib/load.go:42 (setLoadReaderError or equivalent) appends
-- `TracebackFromLastError` to load's error return value. But load returns
-- (nil, errstring) — the error is returned, not raised — so the caller
-- expects a bare diagnostic. Reference Lua leaves the message untouched
-- in this path.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   local f, err = load(coroutine.yield)  -- inside a coroutine
--   print(err)  -> "attempt to yield across a C-call boundary"
--
-- golua today:
--   print(err)  -> "attempt to yield across a C-call boundary
--                   stack traceback:
--                       [C]: in function 'coroutine.yield'
--                       [C]: in global 'load'
--                       ..."
--
-- Discovered: differential fuzz 2026-05-04 (coroutines wave-2 agent).
-- Fix: do not append traceback for load's returned err — only raise+wrap
-- it when load itself raises (which is rare; load normally returns nil+err).

local co = coroutine.create(function()
  local f, err = load(coroutine.yield)
  return err
end)
local ok, err = coroutine.resume(co)
assert(ok, "resume should succeed; got " .. tostring(err))
assert(type(err) == "string", "load should return string error")
assert(not err:find("stack traceback"),
  "load() return-value error must not include 'stack traceback'; got:\n" .. err)
assert(err == "attempt to yield across a C-call boundary",
  "expected exact message 'attempt to yield across a C-call boundary'; got:\n" .. err)

print("ok")
