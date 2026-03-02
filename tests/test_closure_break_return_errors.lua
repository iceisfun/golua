-- Test: closure.lua - Closure x break x return x errors
-- From: closure.lua
-- What: Tests closure behavior across break, return, and error conditions in while loops

do
  local b
  function f(x)
    local first = 1
    while 1 do
      if x == 3 and not first then return end
      local a = 'xuxu'
      b = function (op, y)
            if op == 'set' then a = x+y
            else return a end
          end
      if x == 1 then do break end
      elseif x == 2 then return
      else if x ~= 3 then error() end
      end
      first = nil
    end
  end

  for i=1,3 do
    f(i)
    assert(b('get') == 'xuxu')
    b('set', 10); assert(b('get') == 10+i)
  end
end
