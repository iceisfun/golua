-- Test: nextvar.lua - Large table iteration with pairs
-- From: nextvar.lua
-- What: Tests creating and iterating a 9000-element table with string keys using pairs, verifying all key-value pairs.

do
  local a = {}
  for i=0,10000 do
    if math.fmod(i,10) ~= 0 then
      a['x'..i] = i
    end
  end

  local n = {n=0}
  for i,v in pairs(a) do
    n.n = n.n+1
    assert(i and v and a[i] == v)
  end
  assert(n.n == 9000)
  a = nil
end
