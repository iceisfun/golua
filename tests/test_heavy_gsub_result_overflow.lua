-- Test: a gsub whose result exceeds the string size cap is a CATCHABLE error.
-- What: gsub builds its result in a strings.Builder with no size limit, so a
--       match-heavy substitution with a large replacement (here ~2GB) drove the
--       builder past what Go can allocate and triggered an UNCATCHABLE runtime
--       fatal OOM that aborted the host process — a sandbox escape. gsub now
--       caps the accumulated result at the same 1<<30 limit string.rep/concat
--       use, turning it into a catchable Lua error.
-- Heavy: builds toward ~1GB before the cap fires; gated behind -full.

do
  local ok, msg = pcall(function()
    -- 2000 matches x ~1MB replacement = ~2GB result, over the 1<<30 cap.
    return (("a"):rep(2000)):gsub("a", ("x"):rep(1000000))
  end)
  assert(not ok, "over-cap gsub must error, not succeed or crash the host")
  assert(string.find(msg, "resulting string too large")
      or string.find(msg, "not enough memory")
      or string.find(msg, "string length overflow"),
      "expected a catchable size error, got: " .. tostring(msg))
  print("over-cap gsub caught: " .. tostring(msg))
  print('+')
end
