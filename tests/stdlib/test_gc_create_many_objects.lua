-- Test: gc.lua - Creating many objects
-- From: gc.lua
-- What: Tests creating many tables, strings with gsub, and dynamically loaded functions.
--       Pure Lua stress test of object creation, string operations, and load().

do
  print("creating many objects")

  local limit = 5000

  for i = 1, limit do
    local a = {}; a = nil
  end

  local a = "a"

  for i = 1, limit do
    a = i .. "b";
    a = string.gsub(a, '(%d%d*)', "%1 %1")
    a = "a"
  end

  a = {}

  function a:test ()
    for i = 1, limit do
      load(string.format("function temp(a) return 'a%d' end", i), "")()
      assert(temp() == string.format('a%d', i))
    end
  end

  a:test()
  _G.temp = nil
end
