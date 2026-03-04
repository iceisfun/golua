-- Test: errors.lua - Float-to-integer conversion errors
-- From: errors.lua
-- What: Tests error messages for invalid float-to-integer conversions in bitwise/shift operations

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage("local a = 2.0^100; x = a << 2", "local a")
  checkmessage("local a = 1 >> 2.0^100", "has no integer representation")
  checkmessage("local a = 2.0^100 & 1", "has no integer representation")
  checkmessage("return 6e40 & 7", "has no integer representation")
  checkmessage("return ~-3.009", "has no integer representation")
end
