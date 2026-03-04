-- Test: db.lua - debug.getinfo basics
-- From: db.lua
-- What: Tests debug.getinfo with various options, including C functions, Lua functions, and invalid options

do
  assert(not pcall(debug.getinfo, print, "X"))
  assert(not pcall(debug.getinfo, 0, ">"))
  assert(not debug.getinfo(1000))
  assert(not debug.getinfo(-1))
  local a = debug.getinfo(print)
  assert(a.what == "C" and a.short_src == "[C]")
  a = debug.getinfo(print, "L")
  assert(a.activelines == nil)
end
