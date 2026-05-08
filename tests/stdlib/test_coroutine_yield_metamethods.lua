-- Test: coroutine.lua - Yields inside metamethods
-- From: coroutine.lua
-- What: Tests that coroutine.yield works from within metamethods (__eq, __lt, __add, __sub, __mul, etc.)

do
  local function val(x)
    if type(x) == "table" then return x.x else return x end
  end

  local function new(v)
    return setmetatable({x = v}, mt)
  end

  local function run(f, expectedYields)
    local co = coroutine.wrap(f)
    local i = 1
    while true do
      -- Use table.pack so a leading-nil yield like (nil, "add") is not
      -- truncated by Lua 5.5's first-hole # semantics.
      local res = table.pack(co())
      if res.n == 0 then break end
      if res[1] == nil and res[2] then
        -- it's a yield
        assert(res[2] == expectedYields[i],
               "expected yield '" .. tostring(expectedYields[i]) ..
               "' but got '" .. tostring(res[2]) .. "'")
        i = i + 1
      else
        return res[1]
      end
    end
  end

  mt = {
    __eq = function(a,b) coroutine.yield(nil, "eq"); return val(a) == val(b) end,
    __lt = function(a,b) coroutine.yield(nil, "lt"); return val(a) < val(b) end,
    __add = function(a,b) coroutine.yield(nil, "add"); return val(a) + val(b) end,
    __sub = function(a,b) coroutine.yield(nil, "sub"); return val(a) - val(b) end,
    __mul = function(a,b) coroutine.yield(nil, "mul"); return val(a) * val(b) end,
    __div = function(a,b) coroutine.yield(nil, "div"); return val(a) / val(b) end,
    __pow = function(a,b) coroutine.yield(nil, "pow"); return val(a) ^ val(b) end,
    __mod = function(a,b) coroutine.yield(nil, "mod"); return val(a) % val(b) end,
    __idiv = function(a,b) coroutine.yield(nil, "idiv"); return val(a) // val(b) end,
    __unm = function(a) coroutine.yield(nil, "unm"); return -val(a) end,
    __bnot = function(a) coroutine.yield(nil, "bnot"); return ~val(a) end,
    __shl = function(a,b) coroutine.yield(nil, "shl"); return val(a) << val(b) end,
    __shr = function(a,b) coroutine.yield(nil, "shr"); return val(a) >> val(b) end,
    __band = function(a,b) coroutine.yield(nil, "band"); return val(a) & val(b) end,
    __bor = function(a,b) coroutine.yield(nil, "bor"); return val(a) | val(b) end,
    __bxor = function(a,b) coroutine.yield(nil, "bxor"); return val(a) ~ val(b) end,
    __concat = function(a,b) coroutine.yield(nil, "concat"); return val(a) .. val(b) end,
    __index = function(t,k) coroutine.yield(nil, "index"); return val(t) end,
    __newindex = function(t,k,v) coroutine.yield(nil, "newindex"); end,
  }

  local a = new(10)
  local b = new(12)
  assert(run(function () return a + b end, {"add"}) == 22)
  assert(run(function () return a - b end, {"sub"}) == -2)
  assert(run(function () return a * b end, {"mul"}) == 120)
end
