-- Tail call function name resolution in traceback.
-- When a global function is called via tail call, lua5.4 resolves the
-- function name in the traceback (e.g. "in function 'gfunc'").
-- golua currently shows the anonymous form "in function <file:line>".

-- Global function called via tail call
function gfunc() error("test") end
function caller() return gfunc() end

local ok, tb = xpcall(caller, debug.traceback)
-- The traceback should contain the function name 'gfunc', not <...:line>
print(tb:find("in function 'gfunc'") ~= nil)
--> =true

-- Non-tail call (should always show name — baseline)
function caller2() local r = gfunc(); return r end
local ok2, tb2 = xpcall(caller2, debug.traceback)
print(tb2:find("in function 'gfunc'") ~= nil)
--> =true

-- Tail call chain with (...tail calls...) marker
function chain_a() error("chain") end
function chain_b() return chain_a() end
function chain_c() return chain_b() end
local ok3, tb3 = xpcall(chain_c, debug.traceback)
print(tb3:find("in function 'chain_a'") ~= nil)
--> =true
print(tb3:find("tail calls") ~= nil)
--> =true
