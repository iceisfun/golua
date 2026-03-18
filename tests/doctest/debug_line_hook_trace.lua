-- Test debug line hook traces match Lua 5.4 behavior.
-- Tests VARARGPREP suppression, backward jump detection, CLOSURE line
-- assignment, and per-frame lastHookLine save/restore.

local debug = require "debug"

local function test(s, expected)
  local lines = {}
  local function f(event, line)
    lines[#lines+1] = line
  end
  debug.sethook(f, "l"); load(s)(); debug.sethook()
  local got = table.concat(lines, ",")
  assert(got == expected,
    "wrong trace: expected={" .. expected .. "} got={" .. got .. "}")
end

-- if/else with multi-line condition
test([[if
math.sin(1)
then
  a=1
else
  a=2
end
]], "2,3,4,7")
-->

-- local function definition (CLOSURE on 'end' line)
test([[local function foo()
end
foo()
A = 1
A = 2
A = 3
]], "2,3,2,4,5,6")
_G.A = nil
-->

-- if nil (constant false optimization)
test([[--
if nil then
  a=1
else
  a=2
end
]], "2,5,6")
-->

-- repeat/until loop
test([[a=1
repeat
  a=a+1
until a==3
]], "1,3,4,3,4")
-->

-- do return end
test([[ do
  return
end
]], "2")
-->

-- while loop with backward jump
test([[local a
a=1
while a<=3 do
  a=a+1
end
]], "1,2,3,4,3,4,3,4,3,5")
-->

-- while with break
test([[while math.sin(1) do
  if math.sin(1)
  then break
  end
end
a=1]], "1,2,3,6")
-->

-- numeric for loop
test([[for i=1,3 do
  a=i
end
]], "1,2,1,2,1,2,1,3")
-->

-- generic for loop
test([[for i,v in pairs{'a','b'} do
  a=tostring(i) .. v
end
]], "1,2,1,2,1,3")
-->

-- single-line for loop (backward jump fires each iteration)
test([[for i=1,4 do a=1 end]], "1,1,1,1")
-->

_G.a = nil
print("all line hook trace tests passed")
--> all line hook trace tests passed
