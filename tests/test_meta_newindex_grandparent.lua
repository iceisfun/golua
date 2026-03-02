-- Test: events.lua - __newindex delegation to grandparent (5.1 bug)
-- From: events.lua
-- What: Tests __newindex chaining through parent to grandparent (was a crash in 5.1)

do
  local T, K, V = nil
  local grandparent = {}
  grandparent.__newindex = function(t,k,v) T=t; K=k; V=v end

  local parent = {}
  parent.__newindex = parent
  setmetatable(parent, grandparent)

  local child = setmetatable({}, parent)
  child.foo = 10      --> CRASH (on some machines in 5.1)
  assert(T == parent and K == "foo" and V == 10)
end
