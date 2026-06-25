-- Test: 2-operand concat fast-path overflow is a CATCHABLE error.
-- What: `s = s .. s` doubling goes through the 2-operand OP_CONCAT fast path,
--       which previously had no size guard (unlike the multi-operand path) and
--       drove the result past Go's allocation limit, triggering an UNCATCHABLE
--       runtime fatal OOM that aborted the host process — a sandbox escape.
--       The guard makes it a catchable Lua error. Reference Lua catches the
--       equivalent ("not enough memory"); golua reports "string length overflow"
--       (the same message its other concat paths use).
-- Heavy: builds ~1GB before the guard fires; gated behind -full.

do
  local s = "x"
  local ok, msg = pcall(function()
    while true do
      s = s .. s        -- 2-operand fast path (the regressing path)
    end
  end)
  assert(not ok, "doubling concat must error, not succeed or crash the host")
  assert(string.find(msg, "string length overflow")
      or string.find(msg, "not enough memory"),
      "expected a catchable size/memory error, got: " .. tostring(msg))
  print("2-operand concat overflow caught: " .. tostring(msg))
  print('+')
end
