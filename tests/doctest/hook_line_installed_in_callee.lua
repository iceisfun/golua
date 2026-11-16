-- A line hook installed while a callee runs must not report the caller's
-- line again when the call returns to it: like rethook setting L->oldpc,
-- the caller's pending CALL is the last instruction the hook has seen.
local L = {}
local function h(e, l) L[#L + 1] = l end
local function start()
  debug.sethook(h, "l")
end
local x = 1; start(); local y = 2
local z = 3
debug.sethook()
print(table.concat(L, " "))
--> =8 10 11

-- the same through pcall, a native caller
L = {}
local ok = pcall(function() debug.sethook(h, "l") local a = 1 end); local q = 1
debug.sethook()
print(table.concat(L, " "))
--> =18

-- and through a metamethod
L = {}
local mt = setmetatable({}, {__index = function(t, k) debug.sethook(h, "l") return k end})
local v = mt.foo; local w = 2
local u = 3
debug.sethook()
print(table.concat(L, " "))
--> =26 27

-- with a count hook too, the line event is reported again after each count
-- event, since the count hook's return also resets the tracking
L = {}
debug.sethook(function(e, l) L[#L + 1] = e:sub(1, 1) .. (l or "") end, "l", 1)
local a = 1
local b = 2; local c = 3
debug.sethook()
print(table.concat(L, " "))
--> =c l35 c l36 c l36 c l37 c l37 c l37
