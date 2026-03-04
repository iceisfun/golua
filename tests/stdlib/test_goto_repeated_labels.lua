-- Test: goto.lua - Repeated label errors
-- From: goto.lua
-- What: Tests that defining the same label twice in the same scope is a compile error

do
  local function errmsg (code, m)
    local st, msg = load(code)
    assert(not st and string.find(msg, m))
  end

  errmsg([[ ::l1:: ::l1:: ]], "label 'l1'")
  errmsg([[ ::l1:: do ::l1:: end]], "label 'l1'")
end
