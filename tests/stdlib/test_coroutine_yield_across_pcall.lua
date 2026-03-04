-- Test: coroutine.lua - Yielding across C boundaries
-- From: coroutine.lua
-- What: Tests that yielding from inside pcall/xpcall works (or fails gracefully)

do
  local co = coroutine.wrap(function()
    assert(not pcall(table.sort,{1,2,3}, coroutine.yield))
    assert(coroutine.isyieldable())
    coroutine.yield(20)
    return 30
  end)
  assert(co() == 20)
  assert(co() == 30)
end
