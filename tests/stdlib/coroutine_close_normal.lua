-- coroutine_close_normal.lua
-- coroutine.close() on a "normal" coroutine should raise an error.
-- A "normal" coroutine is one that has resumed another coroutine and is
-- waiting for it to yield/return. It cannot be closed.
-- GoLua previously deadlocked instead of raising an error.

-- Test 1: closing a normal coroutine raises an error (not deadlock)
local co1, co2

co2 = coroutine.create(function()
  -- At this point co1 is in "normal" state (it resumed us).
  assert(coroutine.status(co1) == "normal",
    "expected co1 to be normal, got " .. coroutine.status(co1))

  local ok, msg = pcall(coroutine.close, co1)
  assert(ok == false,
    "expected pcall to fail for normal coroutine, got " .. tostring(ok))
  assert(type(msg) == "string" and msg:find("normal"),
    "expected error mentioning 'normal', got: " .. tostring(msg))
end)

co1 = coroutine.create(function()
  coroutine.resume(co2)
end)

local ok, val = coroutine.resume(co1)
assert(ok, "resume should succeed: " .. tostring(val))

-- Test 2: closing a running coroutine raises an error
local co3 = coroutine.create(function()
  local ok2, msg2 = pcall(coroutine.close, coroutine.running())
  assert(ok2 == false,
    "expected pcall to fail for running coroutine")
  assert(type(msg2) == "string" and msg2:find("running"),
    "expected error mentioning 'running', got: " .. tostring(msg2))
end)
coroutine.resume(co3)

-- Test 3: closing a dead coroutine succeeds
local co4 = coroutine.create(function() end)
coroutine.resume(co4)
assert(coroutine.status(co4) == "dead")
local ok3 = coroutine.close(co4)
assert(ok3 == true, "close dead coroutine should return true")

-- Test 4: closing a suspended coroutine succeeds
local co5 = coroutine.create(function() coroutine.yield() end)
coroutine.resume(co5)
assert(coroutine.status(co5) == "suspended")
local ok4 = coroutine.close(co5)
assert(ok4 == true, "close suspended coroutine should return true")
assert(coroutine.status(co5) == "dead",
  "closed coroutine should be dead, got " .. coroutine.status(co5))

-- Test 5: closing a never-started suspended coroutine succeeds
local co6 = coroutine.create(function() end)
assert(coroutine.status(co6) == "suspended")
local ok5 = coroutine.close(co6)
assert(ok5 == true, "close never-started coroutine should return true")

-- Test 6: verify coroutine.status returns correct values for all states
local status_log = {}
local inner = coroutine.create(function()
  table.insert(status_log, "inner sees outer: " .. coroutine.status(co1))
  coroutine.yield()
end)

local outer
outer = coroutine.create(function()
  table.insert(status_log, "before resume: " .. coroutine.status(inner))
  coroutine.resume(inner)
  table.insert(status_log, "after resume: " .. coroutine.status(inner))
end)
co1 = outer  -- reuse co1 for the inner check

table.insert(status_log, "before start: " .. coroutine.status(outer))
coroutine.resume(outer)
table.insert(status_log, "after finish: " .. coroutine.status(outer))

assert(status_log[1] == "before start: suspended", status_log[1])
assert(status_log[2] == "before resume: suspended", status_log[2])
assert(status_log[3] == "inner sees outer: normal", status_log[3])
assert(status_log[4] == "after resume: suspended", status_log[4])
assert(status_log[5] == "after finish: dead", status_log[5])
