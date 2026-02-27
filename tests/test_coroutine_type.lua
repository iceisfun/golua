-- Bug: Coroutines have type "table" instead of "thread".
-- type(coroutine.create(f)) should return "thread", not "table".

local co = coroutine.create(function() end)
local t = type(co)
assert(t == "thread", "type(coroutine) should be 'thread', got '" .. t .. "'")

-- tostring should show "thread: 0x..." not "table: 0x..."
local s = tostring(co)
assert(s:find("^thread:"), "tostring(coroutine) should start with 'thread:', got '" .. s .. "'")

-- coroutine.running() should also return a thread
local running = coroutine.running()
if running then
  assert(type(running) == "thread", "coroutine.running() type should be 'thread', got '" .. type(running) .. "'")
end

print("PASS")
