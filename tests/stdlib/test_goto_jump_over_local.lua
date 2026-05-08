-- Test: goto.lua - Jumping over variable definition errors
-- From: goto.lua
-- What: Tests that goto cannot jump over a local variable definition

do
  local function errmsg (code, m)
    local st, msg = load(code)
    assert(not st and string.find(msg, m))
  end

  errmsg([[ goto l1; local aa ::l1:: ::l2:: print(3) ]], "scope of 'aa'")

  errmsg([[
do local bb, cc; goto l1; end
local aa
::l1:: print(3)
]], "scope of 'aa'")
end
