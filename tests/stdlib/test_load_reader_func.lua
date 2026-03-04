-- Test: calls.lua - Generic load with reader function
-- From: calls.lua
-- What: Tests load() with a reader function, binary/text mode restrictions

do
  local x = "-- a comment\0\0\0\n  x = 10 + \n23; \
       local a = function () x = 'hi' end; \
       return '\0'"
  local function read1 (x)
    local i = 0
    return function ()
      collectgarbage()
      i=i+1
      return string.sub(x, i, i)
    end
  end

  local a = assert(load(read1(x), "modname", "t", _G))
  assert(a() == "\0" and _G.x == 33)
  assert(debug.getinfo(a).source == "modname")
end
