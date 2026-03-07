-- Test: gc.lua - Functions with errors (substrings of invalid program)
-- From: gc.lua
-- What: Tests that pcall(load(...)) handles partial/invalid Lua code substrings
--       gracefully without crashing. Exercises the parser and error handling.

do
  local prog = [[
do
  a = 10;
  function foo(x,y)
    a = sin(a+0.456-0.23e-12);
    return function (z) return sin(%x+z) end
  end
  local x = function (w) a=a+w; end
end
]]
  local step = 13  -- use larger step to keep runtime manageable
  for i=1, string.len(prog), step do
    for j=i, string.len(prog), step do
      pcall(load(string.sub(prog, i, j), ""))
    end
  end
  rawset(_G, "a", nil)
  _G.x = nil
end
