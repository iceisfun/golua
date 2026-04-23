-- broken_fuzz_traceback_name_labels: missing nameWhat cases in traceback formatting
--
-- BROKEN: vm/vm_debug.go formats traceback frames based on `nameWhat` but only
-- handles a subset of the values. Lua 5.5 reports "in global", "in field",
-- "in method", "in local", "in upvalue"; golua collapses most to
-- "in function '<name>'". Affects both Lua-frame and C-frame formatting.
--
-- Reference (lua5.5.0):
--   Lua frame calling a global function   → "in global 'gfn'"
--   C frame for a global (e.g. xpcall)    → "[C]: in global 'xpcall'"
--   C frame for a table field (e.g. coroutine.yield) → "[C]: in field 'yield'"
--
-- golua today:
--   All of the above collapse to "in function '...'"
--
-- Discovered: differential fuzz 2026-04-23 (debuglib_1 + debuglib_2).
-- This was listed in project_v2_release.md as a partial gap; fuzzing confirmed
-- it extends to C frames and specifically to the "global" nameWhat case.

-- gfn is looked up through _ENV when called by name at chunk scope, so the
-- frame's nameWhat is "global". If we pass gfn as a value to xpcall, the
-- lookup is lost and the frame label degrades to "function" (this is the
-- same in reference Lua 5.5). So we must call gfn by name.
local function inner() error("msg") end
function gfn() inner() end

-- Wrapper so xpcall gets a function value; the gfn() call *inside* wrapper
-- is the one with nameWhat="global".
local ok, tb = xpcall(function() gfn() end, debug.traceback)
assert(not ok)
assert(tostring(tb):find("in global 'gfn'"),
       "expected 'in global gfn', got traceback:\n" .. tostring(tb))

-- C-frame globals: `error` is looked up via _ENV, so nameWhat="global".
assert(tostring(tb):find("%[C%]: in global 'error'"),
       "expected '[C]: in global error', got:\n" .. tostring(tb))

-- C-frame field: coroutine.yield is called as a table access, so nameWhat="field".
-- Force the traceback to include it by yielding and then asking for a traceback
-- on the suspended coroutine.
local co = coroutine.create(function() coroutine.yield() end)
coroutine.resume(co)
local tb3 = debug.traceback(co)
assert(tostring(tb3):find("in field 'yield'"),
       "expected '[C]: in field yield', got:\n" .. tostring(tb3))
