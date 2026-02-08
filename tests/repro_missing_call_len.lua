print("Testing __len...")
local len_mt = {
  __len = function(t)
    return 42
  end
}
local t1 = setmetatable({}, len_mt)
local status, res = pcall(function() return #t1 end)
if status and res == 42 then
  print("PASS: __len works")
else
  print("FAIL: __len failed: " .. tostring(res))
end

print("Testing __call...")
local call_mt = {
  __call = function(t, a, b)
    return (a or 0) + (b or 0) + 10
  end
}
local t2 = setmetatable({}, call_mt)
local status, res = pcall(function() return t2(1, 2) end)
if status and res == 13 then
  print("PASS: __call works")
else
  print("FAIL: __call failed: " .. tostring(res))
end

print("Testing __call with self...")
local call_self_mt = {
  __call = function(t, val)
    if t == nil then return "t is nil" end
    return "called " .. tostring(val)
  end
}
local t3 = setmetatable({}, call_self_mt)
local status, res = pcall(function() return t3("hello") end)
if status and res == "called hello" then
    print("PASS: __call received self")
else
    print("FAIL: __call self check: " .. tostring(res))
end
