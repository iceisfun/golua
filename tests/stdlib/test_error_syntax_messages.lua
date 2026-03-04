-- Test: errors.lua - Syntax error messages
-- From: errors.lua
-- What: Tests that syntax errors include correct line numbers and token information

do
  local function checksyntax (code, expectedmsg, expectedtoken, expectedline)
    local f, msg = load(code)
    assert(not f)
    assert(string.find(msg, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. msg)
    if expectedtoken then
      assert(string.find(msg, expectedtoken, 1, true),
             "expected token '" .. expectedtoken .. "' in: " .. msg)
    end
    if expectedline then
      assert(string.find(msg, ":" .. expectedline .. ":", 1, true),
             "expected line " .. expectedline .. " in: " .. msg)
    end
  end

  checksyntax([[
  local a = {4

]], "'}' expected (to close '{' at line 1)", "<eof>", 3)
end
