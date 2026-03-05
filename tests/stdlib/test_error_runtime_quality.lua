-- Test: errors.lua - Runtime error message quality
-- From: errors.lua
-- What: Tests that runtime errors include useful context (variable names, operation types)

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage("a = {} + 1", "arithmetic")
  checkmessage("a = {} | 1", "bitwise operation")
  checkmessage("a = {} < 1", "attempt to compare")
  checkmessage("aaa=1; bbbb=2; aaa=math.sin(3)+bbbb(3)", "global 'bbbb'")
  checkmessage("local a={}; a.bbbb(3)", "field 'bbbb'")
  checkmessage("local a; a(13)", "local 'a'")
end
