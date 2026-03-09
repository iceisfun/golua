-- Sparse constructor list fields should keep a Lua 5.4-compatible border.

-- Original bug: table.concat on sparse constructor returned "" instead of erroring.
do
  local t = {nil, nil, {}, 2.5, nil}
  assert(#t == 4, "expected #t == 4, got " .. tostring(#t))

  local ok, err = pcall(table.concat, t, "")
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "invalid value (nil) at index 1", 1, true), err)
end

-- Constructor border patterns: last non-nil determines #.
do
  assert(#{nil} == 0)
  assert(#{nil, nil} == 0)
  assert(#{1, nil} == 1)
  assert(#{nil, 1} == 2)
  assert(#{nil, 1, nil} == 2)
  assert(#{1, nil, nil} == 1)
  assert(#{nil, nil, 1} == 3)
  assert(#{1, nil, 2, nil} == 3)
  assert(#{1, nil, nil, nil, 5} == 5)
  assert(#{nil, nil, nil, nil, 5, nil} == 5)
end

-- Dense (no-hole) tables: fast path still works.
do
  assert(#{1, 2, 3} == 3)
  assert(#{"a"} == 1)
  assert(#{} == 0)
end
