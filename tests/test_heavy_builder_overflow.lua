-- Test: unbounded string/table builders cap their result instead of crashing.
-- What: string.pack fixed-size directives and table.concat built results with no
--       Go-safe size limit, so a huge `c<N>` directive or a concat of large
--       elements grew past what Go can allocate and triggered an UNCATCHABLE
--       runtime fatal OOM that aborted the host — a sandbox escape. Both now cap
--       at the same 1<<30 limit string.rep/concat/gsub use, raising a catchable
--       Lua error.
-- The string.pack case is instant (the cap fires on the size check before any
-- allocation); the table.concat case is heavy (builds real >1GB content), so the
-- whole file is gated behind -full.

do
  -- string.pack: a fixed-size directive over the 1<<30 cap (but under the 2^31
  -- size-digit limit, so it reaches the cap rather than a parse error) is
  -- rejected before allocation.
  local ok, msg = pcall(function() return string.pack("c1500000000", "x") end)
  assert(not ok and string.find(msg, "result too long"),
    "pack over-cap must be a catchable 'result too long', got: " .. tostring(msg))
  print("pack over-cap caught: " .. tostring(msg))

  -- table.concat: a joined result over the cap is a catchable error, not a crash.
  local a = ("x"):rep(600000000) -- 600MB
  local ok2, msg2 = pcall(function() return table.concat({a, a}) end) -- 1.2GB > 1<<30
  assert(not ok2 and (string.find(msg2, "resulting string too large")
      or string.find(msg2, "not enough memory")),
    "table.concat over-cap must be catchable, got: " .. tostring(msg2))
  print("concat over-cap caught: " .. tostring(msg2))
  print('+')
end
