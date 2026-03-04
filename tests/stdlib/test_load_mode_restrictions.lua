-- Test: calls.lua - Load mode restrictions
-- From: calls.lua
-- What: Tests that load() properly rejects text in binary mode and binary in text mode

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

  local function cannotload (msg, a,b)
    assert(not a and string.find(b, msg))
  end

  cannotload("attempt to load a text chunk", load(read1(x), "modname", "b", {}))
  cannotload("attempt to load a text chunk", load(x, "modname", "b"))
end
