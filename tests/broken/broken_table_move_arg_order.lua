-- table.move() with no args should check arg #2 (first number) first,
-- not arg #1 (table). Lua 5.4's luaB_move checks numeric args before table.
local ok, err

ok, err = pcall(table.move)
assert(err == "bad argument #2 to 'table.move' (number expected, got no value)",
  "expected arg #2 check first, got: " .. tostring(err))

-- Even with a non-table first arg, the number arg is checked first
ok, err = pcall(table.move, "hello")
assert(err == "bad argument #2 to 'table.move' (number expected, got no value)",
  "expected arg #2 check even with string first arg, got: " .. tostring(err))

print("OK")
