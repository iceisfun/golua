-- Test: warn() argument validation
-- From: main.lua
-- What: Tests that warn() requires at least one string argument and does not leave unfinished warnings on error.

do
  -- 'warn' must get at least one argument
  local st, msg = pcall(warn)
  assert(string.find(msg, "string expected"))

  -- 'warn' does not leave unfinished warning in case of errors
  -- (message would appear in next warning)
  st, msg = pcall(warn, "SHOULD NOT APPEAR", {})
  assert(string.find(msg, "string expected"))
end
