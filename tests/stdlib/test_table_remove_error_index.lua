-- table.remove position range errors are reported on argument #2 in Lua 5.4.

do
  local ok, err

  ok, err = pcall(table.remove, {"a", nil}, -1)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)
  assert(string.find(err, "position out of bounds", 1, true), err)

  ok, err = pcall(table.remove, {}, 2)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)
  assert(string.find(err, "position out of bounds", 1, true), err)
end
