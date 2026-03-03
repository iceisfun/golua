-- Test: attrib.lua - _ENV const assignment error
-- From: attrib.lua
-- What: Assigning to _ENV when declared const should produce an error

do
  local function foo ()
    local _ENV <const> = 11
    X = "hi"
  end
  local st, msg = pcall(foo)
  assert(not st and string.find(msg, "number"))
end
