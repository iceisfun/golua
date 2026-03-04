-- Test: nextvar.lua - Table library with metamethods
-- From: nextvar.lua
-- What: Tests table.insert, table.remove, table.sort, table.concat, and table.unpack through metatable proxies (__len, __index, __newindex).

do   -- testing table library with metamethods
  local function test (proxy, t)
    for i = 1, 10 do
      table.insert(proxy, 1, i)
    end
    assert(#proxy == 10 and #t == 10 and proxy[1] ~= undef)
    for i = 1, 10 do
      assert(t[i] == 11 - i)
    end
    table.sort(proxy)
    for i = 1, 10 do
      assert(t[i] == i and proxy[i] == i)
    end
    assert(table.concat(proxy, ",") == "1,2,3,4,5,6,7,8,9,10")
    for i = 1, 8 do
      assert(table.remove(proxy, 1) == i)
    end
    assert(#proxy == 2 and #t == 2)
    local a, b, c = table.unpack(proxy)
    assert(a == 9 and b == 10 and c == nil)
  end

  -- all virtual
  local t = {}
  local proxy = setmetatable({}, {
    __len = function () return #t end,
    __index = t,
    __newindex = t,
  })
  test(proxy, t)

  -- only __newindex
  local count = 0
  t = setmetatable({}, {
    __newindex = function (t,k,v) count = count + 1; rawset(t,k,v) end})
  test(t, t)
  assert(count == 10)   -- after first 10, all other sets are not new

  -- no __newindex
  t = setmetatable({}, {
    __index = function (_,k) return k + 1 end,
    __len = function (_) return 5 end})
  assert(table.concat(t, ";") == "2;3;4;5;6")

end
