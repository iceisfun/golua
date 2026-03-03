-- Test: errors.lua - Metamethod error messages
-- From: errors.lua
-- What: Tests error messages when metamethods are not callable

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage([[
  local a = setmetatable({}, {__add = 34})
  a = a + 1
]], "metamethod 'add'")
  checkmessage([[
  local a = setmetatable({}, {__lt = {}})
  a = a > a
]], "metamethod 'lt'")
end
