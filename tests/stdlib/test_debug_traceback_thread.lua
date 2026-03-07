-- debug.traceback(thread, msg, level) should return a string containing the
-- traceback of the given coroutine, not the thread object itself.
--
-- Lua 5.4 Reference §6.10: "If message is present but is neither a string
-- nor nil, this function returns message without further processing."
-- When a thread is the first argument, it generates the traceback for that thread.

-- Test 1: basic traceback of suspended coroutine
local co = coroutine.create(function()
  coroutine.yield()
end)
coroutine.resume(co)

local result = debug.traceback(co, "test-msg", 0)
assert(type(result) == "string",
  "debug.traceback(thread) should return string, got: " .. type(result))
assert(string.find(result, "test%-msg"),
  "traceback should contain message, got: " .. result)

-- Test 2: traceback with nil message
local result2 = debug.traceback(co, nil, 0)
assert(type(result2) == "string",
  "debug.traceback(thread, nil) should return string, got: " .. type(result2))

-- Test 3: traceback of coroutine at deeper call stack
local co2 = coroutine.create(function()
  local function inner()
    coroutine.yield()
  end
  inner()
end)
coroutine.resume(co2)

local result3 = debug.traceback(co2, "deep", 0)
assert(type(result3) == "string",
  "deep traceback should return string, got: " .. type(result3))
assert(string.find(result3, "deep"),
  "deep traceback should contain message")

-- Test 4: traceback of dead coroutine
local co3 = coroutine.create(function() end)
coroutine.resume(co3)
assert(coroutine.status(co3) == "dead")

local result4 = debug.traceback(co3, "dead-msg", 0)
assert(type(result4) == "string",
  "dead coroutine traceback should return string, got: " .. type(result4))

