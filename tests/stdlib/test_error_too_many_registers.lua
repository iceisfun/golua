-- Test: errors.lua - Too many registers error
-- From: errors.lua
-- What: Tests that functions with too many arguments produce "too many registers" error

do
  local function checkmessage(code, expectedmsg)
    local f, err = load(code)
    if f then
      local ok
      ok, err = pcall(f)
    end
    assert(err and string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage("a = f(x" .. string.rep(",x", 260) .. ")", "too many registers")
end
