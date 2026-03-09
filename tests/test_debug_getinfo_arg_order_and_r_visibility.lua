-- debug.getinfo argument validation order and transfer visibility parity.

do
  local ok, err

  -- what-arg type validation happens before level coercion in this shape.
  ok, err = pcall(function()
    return debug.getinfo({}, {})
  end)
  assert(not ok)
  assert(string.find(err, "bad argument #2", 1, true), err)
  assert(string.find(err, "string expected, got table", 1, true), err)

  -- Numeric what is coerced to string and then validated as option set.
  ok, err = pcall(function()
    return debug.getinfo(coroutine.running(), 1, 123)
  end)
  assert(not ok)
  assert(string.find(err, "bad argument #3", 1, true), err)
  assert(string.find(err, "invalid option", 1, true), err)

  -- Non-integer numeric level gets the integer-representation error.
  ok, err = pcall(function()
    return debug.getinfo(1.5)
  end)
  assert(not ok)
  assert(string.find(err, "bad argument #1", 1, true), err)
  assert(string.find(err, "number has no integer representation", 1, true), err)

  -- Outside hook callbacks, transfer fields are reported as 0/0.
  local info = debug.getinfo(debug.getinfo, "r")
  assert(type(info) == "table")
  assert(info.ftransfer == 0, "expected ftransfer=0, got " .. tostring(info.ftransfer))
  assert(info.ntransfer == 0, "expected ntransfer=0, got " .. tostring(info.ntransfer))
end
