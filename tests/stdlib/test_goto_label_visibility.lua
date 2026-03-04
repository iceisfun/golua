-- Test: goto.lua - Label visibility errors (cannot see label inside block)
-- From: goto.lua
-- What: Tests that goto cannot jump to a label defined inside a nested block

do
  local function errmsg (code, m)
    local st, msg = load(code)
    assert(not st and string.find(msg, m))
  end

  errmsg([[ goto l1; do ::l1:: end ]], "label 'l1'")
  errmsg([[ do ::l1:: end goto l1; ]], "label 'l1'")
end
