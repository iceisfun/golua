-- Test: tail call hooks should fire as "tail call" not "call"

-- Test 1: coroutine tail call hooks
local events = {}
local co = coroutine.create(function()
    local function a() return 1 end
    local function b() return a() end  -- tail call
    return b()  -- tail call
end)
debug.sethook(co, function(event) events[#events+1] = event end, "cr")
coroutine.resume(co)

local found_tail = 0
local found_call = 0
for _, e in ipairs(events) do
    if e == "tail call" then found_tail = found_tail + 1 end
    if e == "call" then found_call = found_call + 1 end
end
-- Lua 5.4 produces: call, tail call, tail call, return
assert(found_tail == 2, "expected 2 tail call events, got " .. found_tail ..
    " (events: " .. table.concat(events, ", ") .. ")")
assert(found_call == 1, "expected 1 call event, got " .. found_call)

-- Test 2: non-coroutine tail call hooks
events = {}
local function test()
    local function a() return 1 end
    local function b() return a() end
    return b()
end
debug.sethook(function(event) events[#events+1] = event end, "cr")
test()
debug.sethook()

found_tail = 0
for _, e in ipairs(events) do
    if e == "tail call" then found_tail = found_tail + 1 end
end
assert(found_tail == 2, "expected 2 tail call events in non-coroutine test, got " .. found_tail ..
    " (events: " .. table.concat(events, ", ") .. ")")

print("OK")
