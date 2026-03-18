-- Test ftransfer/ntransfer for call/return hooks
local debug = require "debug"

local on = false
local inp, out

local function hook(event)
  if not on then return end
  local ar = debug.getinfo(2, "ruS")
  local t = {}
  for i = ar.ftransfer, ar.ftransfer + ar.ntransfer - 1 do
    local _, v = debug.getlocal(2, i)
    t[#t + 1] = v
  end
  if event == "return" then
    out = t
  else
    inp = t
  end
end

debug.sethook(hook, "cr")

-- Test 1: math.sin
on = true; math.sin(3); on = false
assert(#inp == 1 and inp[1] == 3)
assert(#out == 1 and out[1] == math.sin(3))

-- Test 2: select
on = true; select(2, 10, 20, 30, 40); on = false
assert(#inp == 5)
assert(#out == 3 and out[1] == 20 and out[2] == 30 and out[3] == 40)

-- Test 3: tail call with varargs
local function foo(a, ...) return ... end
local function foo1() on = not on; return foo(20, 10, 0) end
foo1(); on = false
-- ntransfer for call should be 1 (just fixed param 'a')
assert(#inp == 1 and inp[1] == 20, "call: expected {20}, got " .. #inp)
-- return values are the varargs
assert(#out == 2 and out[1] == 10 and out[2] == 0)

debug.sethook()
print("PASS") --> PASS
