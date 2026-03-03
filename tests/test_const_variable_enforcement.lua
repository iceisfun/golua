-- Test: constructs.lua - Const variable enforcement
-- From: constructs.lua
-- What: Tests that const and close variable attributes are properly enforced

do
  local function checkload(code, expectedmsg)
    local f, msg = load(code)
    assert(f == nil and string.find(msg, expectedmsg),
           "expected error '" .. expectedmsg .. "' but got: " .. tostring(msg))
  end

  local prog <const> = [[local x <XXX> = 10]]
  checkload(prog, "unknown attribute 'XXX'")
  checkload([[local xxx <const> = 20; xxx = 10]],
             "attempt to assign to const variable 'xxx'")
  checkload([[
    local x <close> = nil
    x = io.open()
  ]], "attempt to assign to const variable 'x'")
end
