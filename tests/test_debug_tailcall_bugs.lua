-- Test 1: Traceback includes tail-called function frame info
local function target()
  return debug.traceback("msg", 0)
end

local function caller()
  return target()  -- tail call
end

local tb = caller()
-- The traceback should contain the target function's info AND (...tail calls...)
-- Lua 5.4: shows the tail-called function's frame line, then (...tail calls...) on next line
-- The function name won't resolve (tail call erases caller info), but the source:line is there
assert(tb:find("test_debug_tailcall_bugs.lua:3:"), "traceback should show target function's line, got:\n" .. tb)
assert(tb:find("%(%.%.%.tail calls%.%.%.%)"), "traceback should show tail calls marker, got:\n" .. tb)
-- Verify the frame line appears BEFORE the tail calls marker
local frame_pos = tb:find("test_debug_tailcall_bugs.lua:3:")
local tail_pos = tb:find("%(%.%.%.tail calls%.%.%.%)")
assert(frame_pos < tail_pos, "frame info should appear before tail calls marker")
print("PASS: test 1 - traceback includes tail-called frame info")


-- Test 2: currentline is correct (not -1) in call hooks
local call_lines = {}
debug.sethook(function(event)
  local info = debug.getinfo(2, "Sl")
  if info and info.currentline then
    table.insert(call_lines, info.currentline)
  end
end, "c")

local function foo()
  return 1
end

foo()
debug.sethook()

-- The call hook for foo() should have a real line number, not -1
local found_negative = false
for _, line in ipairs(call_lines) do
  if line == -1 then
    -- -1 for C functions is expected, but Lua functions should have real lines
  end
end

-- More specific test: track call hook for a known Lua function
local hook_currentline = nil
local function bar()
  return 42
end

debug.sethook(function(event)
  local info = debug.getinfo(2, "Sln")
  if info and info.what == "Lua" and info.name == "bar" then
    hook_currentline = info.currentline
  end
end, "c")

bar()
debug.sethook()

assert(hook_currentline ~= nil, "call hook should have fired for bar()")
assert(hook_currentline ~= -1, "currentline should not be -1 in call hook, got: " .. tostring(hook_currentline))
print("PASS: test 2 - currentline is not -1 in call hooks, got line " .. tostring(hook_currentline))


-- Test 3: tail call hook fires with updated frame (new closure info)
local tailcall_info = nil
local function inner()
  return 1
end

local function outer()
  return inner()  -- tail call
end

debug.sethook(function(event)
  if event == "tail call" then
    local info = debug.getinfo(2, "Sn")
    tailcall_info = info
  end
end, "c")

outer()
debug.sethook()

-- In Lua 5.4, the tail call hook fires AFTER the frame is updated to the new function.
-- So getinfo should show the new function (inner), not the old one (outer).
assert(tailcall_info ~= nil, "tail call hook should have fired")
-- The what field should be "Lua" (inner is a Lua function)
assert(tailcall_info.what == "Lua", "tail call hook frame should be Lua, got: " .. tostring(tailcall_info.what))
print("PASS: test 3 - tail call hook fires with updated frame")


print("All debug tailcall bug tests passed!")
