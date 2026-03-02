-- Test: constructs.lua - Invalid operations in dead code
-- From: constructs.lua
-- What: Tests that invalid operations don't raise errors when not executed

do
  if false then a = 3 // 0; a = 0 % 0 end
end
