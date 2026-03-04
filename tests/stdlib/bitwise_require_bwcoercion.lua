-- Test: bitwise.lua - Require bwcoercion helper
-- From: bitwise.lua
-- What: Loads the bitwise coercion helper that adds string metamethods for bitwise ops

do
  require "bwcoercion"
  local numbits = string.packsize('j') * 8
  assert(~0 == -1)
  assert((1 << (numbits - 1)) == math.mininteger)
end
