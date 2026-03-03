-- Test: calls.lua - Multiple return values
-- From: calls.lua
-- What: Tests functions returning multiple values and table.pack/unpack interactions

do
  local function unlpack (t, i)
    i = i or 1
    if (i <= #t) then return t[i], unlpack(t, i+1) end
  end

  local function f() return 1,2,30,4 end
  local function ret2 (a,b) return a,b end

  local a,b,c,d = unlpack{1,2,3}
  assert(a==1 and b==2 and c==3 and d==nil)

  a,b,c,d = ret2(f()), ret2(f())
  assert(a==1 and b==1 and c==2 and d==nil)
end
