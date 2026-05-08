-- Test: goto.lua - Cannot continue repeat-until with variables
-- From: goto.lua
-- What: Tests that goto cannot jump past a local declaration to a label inside repeat-until

do
  local function errmsg (code, m)
    local st, msg = load(code)
    assert(not st and string.find(msg, m))
  end

  errmsg([[
  repeat
    if x then goto cont end
    local xuxu = 10
    ::cont::
  until xuxu < x
]], "scope of 'xuxu'")
end
