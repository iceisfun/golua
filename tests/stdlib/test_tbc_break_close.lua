-- Test: Break generates correct close instruction (5.4.4 bug)
-- From: locals.lua
-- What: Tests that break correctly generates close instructions for to-be-closed variables when leaving a for loop block, fixing a bug from Lua 5.4.4.

do
  local function func2close (f, x, y)
    local obj = setmetatable({}, {__close = f})
    if x then
      return x, obj, y
    else
      return obj
    end
  end

  local closed = false

  local o1 = setmetatable({}, {__close=function() closed = true end})

  local function test()
    for k, v in next, {}, nil, o1 do
      local function f() return k end   -- create an upvalue
      break
    end
    assert(closed)
  end

  test()
end
