-- Test: constructs.lua - Syntax error detection
-- From: constructs.lua
-- What: Tests that specific syntax errors are properly detected

do
  local function checkload(code, expectedmsg)
    local f, msg = load(code)
    assert(f == nil and string.find(msg, expectedmsg),
           "expected error '" .. expectedmsg .. "' but got: " .. tostring(msg))
  end

  checkload("for x do", "expected")
  checkload("x:call", "expected")
end
