-- Test: errors.lua - Named objects (__name metafield)
-- From: errors.lua
-- What: Tests that __name metafield affects tostring and error messages

do
  local function checkmessage(code, expectedmsg)
    local f = assert(load(code))
    local ok, err = pcall(f)
    assert(not ok, "expected error for: " .. code)
    assert(string.find(err, expectedmsg, 1, true),
           "expected '" .. expectedmsg .. "' in: " .. tostring(err))
  end

  checkmessage("math.sin(io.input())", "(number expected, got FILE*)")
  _G.XX = setmetatable({}, {__name = "My Type"})
  assert(string.find(tostring(XX), "^My Type"))
  checkmessage("io.input(XX)", "(FILE* expected, got My Type)")
  checkmessage("return XX + 1", "on a My Type value")
end
