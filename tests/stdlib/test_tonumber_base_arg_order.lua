-- tonumber(v, base): Lua 5.4 validation order is
-- (1) arg2 integer type, (2) arg1 string type, (3) base range.

do
  local ok, err

  -- Both args wrong type: arg2 type error comes first (not arg1).
  ok, err = pcall(tonumber, true, true)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)

  ok, err = pcall(tonumber, 10, true)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)

  -- arg2 is valid integer, arg1 is non-string: arg1 type error.
  ok, err = pcall(tonumber, true, 1)
  assert(not ok)
  assert(string.find(err, "bad argument #1", 1, true), err)
  assert(string.find(err, "string expected, got boolean", 1, true), err)

  ok, err = pcall(tonumber, {}, 37)
  assert(not ok)
  assert(string.find(err, "bad argument #1", 1, true), err)
  assert(string.find(err, "string expected, got table", 1, true), err)

  ok, err = pcall(tonumber, true, 0)
  assert(not ok)
  assert(string.find(err, "bad argument #1", 1, true), err)

  -- arg1 is string, arg2 out of range: arg2 range error.
  ok, err = pcall(tonumber, "10", 1)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)
  assert(string.find(err, "base out of range", 1, true), err)

  -- arg2 is non-numeric string: arg2 type error.
  ok, err = pcall(tonumber, "10", true)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)
end
