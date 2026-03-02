-- Test: Vararg with many fixed parameters
-- From: vararg.lua
-- What: Tests varargs with 4 fixed parameters and extra arguments via {...}.

do
local lim = 20
local i, a = 1, {}
while i <= lim do a[i] = i+0.3; i=i+1 end

function f(a, b, c, d, ...)
  local more = {...}
  assert(a == 1.3 and more[1] == 5.3 and
         more[lim-4] == lim+0.3 and not more[lim-3])
end

local function g (a,b,c)
  assert(a == 1.3 and b == 2.3 and c == 3.3)
end

local call = function (f, args) return f(table.unpack(args, 1, args.n)) end

call(f, a)
call(g, a)
end
