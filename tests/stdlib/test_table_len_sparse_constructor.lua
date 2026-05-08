-- Sparse constructor list fields under Lua 5.5 # semantics: # scans
-- forward from index 1 and returns the index of the first nil (or 0 if
-- t[1] is nil).  Verified against lua5.5.0.

-- Sparse table starting with nil: # returns 0 and table.concat is happy
-- because the empty range [1..0] has nothing to concatenate.
do
  local t = {nil, nil, {}, 2.5, nil}
  assert(#t == 0, "expected #t == 0, got " .. tostring(#t))
  assert(table.concat(t, "") == "")
end

-- Constructor border patterns under 5.5 first-hole semantics.
do
  assert(#{nil} == 0)
  assert(#{nil, nil} == 0)
  assert(#{1, nil} == 1)
  assert(#{nil, 1} == 0)
  assert(#{nil, 1, nil} == 0)
  assert(#{1, nil, nil} == 1)
  assert(#{nil, nil, 1} == 0)
  assert(#{1, nil, 2, nil} == 1)
  assert(#{1, nil, nil, nil, 5} == 1)
  assert(#{nil, nil, nil, nil, 5, nil} == 0)
end

-- Dense (no-hole) tables: fast path still works.
do
  assert(#{1, 2, 3} == 3)
  assert(#{"a"} == 1)
  assert(#{} == 0)
end
