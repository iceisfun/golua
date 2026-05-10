-- test_registry_layout_lua55: registry table layout follows Lua 5.5.
--
-- Lua 5.5 redefined LUA_RIDX_MAINTHREAD from 1 to 3 (lua.h). Slot 1 is now
-- reserved (`false`), slot 2 stays _G, slot 3 holds the main thread.
-- Reference Lua 5.5.0 reports:
--   r[1]  boolean  false
--   r[2]  table    _G
--   r[3]  thread   main thread

local r = debug.getregistry()

assert(r[1] == false, "registry[1] must be false in Lua 5.5; got " .. tostring(r[1]))
assert(r[2] == _G, "registry[2] must be _G; got " .. tostring(r[2]))
assert(type(r[3]) == "thread", "registry[3] must be the main thread; got " .. type(r[3]))

-- Main thread should be the current running thread (coroutine.running() == nil
-- for main, but the registry stores a thread object regardless).
assert(r[3] ~= nil, "main thread slot must not be nil")

print("test_registry_layout_lua55: OK")
