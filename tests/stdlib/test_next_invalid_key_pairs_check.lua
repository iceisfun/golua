-- Test: nextvar.lua - Invalid key to next, pairs/ipairs argument check
-- From: nextvar.lua
-- What: Tests that next raises an error for an invalid key, and that pairs and ipairs require an argument.

do
  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  checkerror("invalid key", next, {10,20}, 3)

  -- both 'pairs' and 'ipairs' need an argument
  checkerror("bad argument", pairs)
  checkerror("bad argument", ipairs)
end
