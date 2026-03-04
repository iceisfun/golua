-- Test: errors.lua - Numeric for loop errors
-- From: errors.lua
-- What: Tests error messages for invalid types in numeric for loop (initial, limit, step)

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage("for i = {}, 10 do end", "table")
  checkmessage("for i = 1, 'x', 10 do end", "string")
  checkmessage("for i = 1, {}, 10 do end", "limit")
  checkmessage("for i = 1, 10, print do end", "step")
end
