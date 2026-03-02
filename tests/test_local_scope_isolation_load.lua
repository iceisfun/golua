-- Test: Local scope isolation via load
-- From: locals.lua
-- What: Tests that locals declared inside a loaded chunk do not leak into the outer environment.

do
  a = nil
  load('local a = {}')()
  assert(a == nil)
end
