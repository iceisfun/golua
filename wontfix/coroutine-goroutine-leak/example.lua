-- golua backs each coroutine with a goroutine. A SUSPENDED coroutine is a
-- goroutine parked on its resume channel. If a suspended coroutine is abandoned
-- (never resumed to completion and never coroutine.close()'d), its goroutine
-- cannot be reaped: Go cannot kill a blocked goroutine, and the parked goroutine
-- is itself a GC root, so neither GC nor the Lua thread becoming unreachable
-- frees it. It leaks until the process exits.
--
-- Reference Lua collects an abandoned suspended coroutine like any other object.

-- LEAKS one goroutine per coroutine (do NOT do this in a long-lived VM):
for i = 1, 1000 do
  local co = coroutine.create(function() for j = 1, 100 do coroutine.yield(j) end end)
  coroutine.resume(co)            -- now suspended, then abandoned
end

-- SUPPORTED patterns — the goroutine is reaped:
do
  local co = coroutine.create(function() for j = 1, 100 do coroutine.yield(j) end end)
  coroutine.resume(co)
  coroutine.close(co)             -- (a) explicit close reaps it
end
do
  local co = coroutine.create(function() return 1 end)
  while coroutine.status(co) ~= "dead" do coroutine.resume(co) end  -- (b) run to completion
end
