-- Test: coroutine.lua - Trivial coroutine errors
-- From: coroutine.lua
-- What: Tests that coroutine.resume and coroutine.status reject non-thread arguments

do
  assert(not pcall(coroutine.resume, 0))
  assert(not pcall(coroutine.status, 0))
end
