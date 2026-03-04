-- Test: db.lua - Function info (nparams, nups, isvararg)
-- From: db.lua
-- What: Tests debug.getinfo "u" option for parameter count, upvalue count, and vararg info

do
  local t = debug.getinfo(print, "u")
  assert(t.isvararg == true and t.nparams == 0 and t.nups == 0)

  t = debug.getinfo(function (a,b,c) end, "u")
  assert(t.isvararg == false and t.nparams == 3 and t.nups == 0)

  t = debug.getinfo(function (a,b,...) return t[a] end, "u")
  assert(t.isvararg == true and t.nparams == 2 and t.nups == 1)
end
