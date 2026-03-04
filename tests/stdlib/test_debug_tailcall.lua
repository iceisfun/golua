-- Test: db.lua - Tail call detection
-- From: db.lua
-- What: Tests that debug.getinfo properly identifies tail calls

do
  local function f (x)
    if x then
      assert(debug.getinfo(1, "S").what == "Lua")
      assert(debug.getinfo(1, "t").istailcall == true)
    end
  end

  function g(x) return f(x) end
  function g1(x) g(x) end
  local function h (x) local f=g1; return f(x) end
  h(true)
end
