-- Mixed constructors should keep keyed fields from changing list border.

do
  local t0 = {{}, [3] = {}, [2] = {}}
  assert(#t0 == 3, "expected #t0 == 3, got " .. tostring(#t0))

  local t = {"a", false, [4] = {}, [5] = nil, [3] = {}}
  t[2] = nil

  -- Lua 5.4 border behavior: #t == 1 for this shape.
  assert(#t == 1, "expected #t == 1, got " .. tostring(#t))

  local ok, err = pcall(table.remove, t, -1)
  assert(not ok)
  assert(type(err) == "string")
  assert(string.find(err, "position out of bounds", 1, true), err)

  local t2 = {1, false, 1, nil, "a", nil, [4] = "x"}
  local removed = table.remove(t2, 5)
  assert(removed == "a")
  assert(#t2 == 3, "expected #t2 == 3 after removal, got " .. tostring(#t2))
end
