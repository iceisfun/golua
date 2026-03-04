-- Test: coroutine.lua - Attempt to resume self
-- From: coroutine.lua
-- What: Tests that a coroutine cannot resume itself

do
  local function co_func (current_co)
    assert(coroutine.running() == current_co)
    assert(coroutine.resume(current_co) == false)
    coroutine.yield(10, 20)
    assert(coroutine.resume(current_co) == false)
    coroutine.yield(23)
    return 10
  end
  local co = coroutine.create(co_func)
  local a,b,c = coroutine.resume(co, co)
  assert(a == true and b == 10 and c == 20)
end
