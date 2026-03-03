-- Test: goto.lua - Jumping into a block errors
-- From: goto.lua
-- What: Tests that goto cannot jump into a block from outside it

do
  local function errmsg (code, m)
    local st, msg = load(code)
    assert(not st and string.find(msg, m))
  end

  errmsg([[ do ::l1:: end goto l1 ]], "label 'l1'")
  errmsg([[ goto l1 do ::l1:: end ]], "label 'l1'")
end
