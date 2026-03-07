-- coroutine.isyieldable(thread) should return true for a suspended coroutine
-- that yielded from a yieldable context.
--
-- Lua 5.4 Reference §6.2: "coroutine.isyieldable([co])" — If co is given,
-- reports whether that coroutine can yield. A suspended coroutine that yielded
-- is in a yieldable state.

-- Test 1: suspended coroutine after yield is yieldable
local co = coroutine.create(function()
  coroutine.yield()
end)
coroutine.resume(co)
assert(coroutine.status(co) == "suspended")
assert(coroutine.isyieldable(co) == true,
  "suspended coroutine after yield should be yieldable, got: " ..
  tostring(coroutine.isyieldable(co)))

-- Test 2: dead coroutine
coroutine.resume(co)
assert(coroutine.status(co) == "dead")

-- Test 3: newly created (never resumed) coroutine is yieldable
local co2 = coroutine.create(function()
  coroutine.yield()
end)
assert(coroutine.status(co2) == "suspended")
assert(coroutine.isyieldable(co2) == true,
  "newly created coroutine should be yieldable, got: " ..
  tostring(coroutine.isyieldable(co2)))

-- Test 4: coroutine suspended inside pcall is still yieldable
local co3 = coroutine.create(function()
  pcall(function()
    coroutine.yield()
  end)
end)
coroutine.resume(co3)
assert(coroutine.status(co3) == "suspended")
assert(coroutine.isyieldable(co3) == true,
  "coroutine yielded inside pcall should be yieldable")

-- Test 5: no-arg isyieldable from main chunk
assert(coroutine.isyieldable() == false,
  "main chunk should not be yieldable")

