-- Line hooks should fire with nil line for stripped functions
local events = {}
local function foo()
  local x = 1
  local y = 2
  return x + y
end
local s = assert(load(string.dump(foo, true)))
debug.sethook(function(e, l)
  if e == "line" then
    events[#events+1] = tostring(l)
  end
end, "l")
s()
debug.sethook()
-- Events: line for sethook return, line:nil inside stripped func, line for debug.sethook()
-- We check that at least one "nil" line event was fired
local has_nil = false
for _, v in ipairs(events) do
  if v == "nil" then has_nil = true end
end
print(has_nil)
--> true
