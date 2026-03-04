-- Test: Malformed number literals
-- From: literals.lua
-- What: Tests that the parser produces proper error messages for malformed number literals.

do
  local function malformednum (n, exp)
    local s, msg = load("return " .. n)
    assert(not s and string.find(msg, exp))
  end

  malformednum("0xe-", "near <eof>")
  malformednum("0xep-p", "malformed number")
  malformednum("1print()", "malformed number")
end
