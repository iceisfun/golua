local function assert_true(cond, msg)
  if not cond then error(msg or "assertion failed") end
end

print("Testing __eq...")
local eq_mt = {
  __eq = function(a, b)
    return true
  end
}
local t1 = setmetatable({}, eq_mt)
local t2 = setmetatable({}, eq_mt)
-- Should be true because of __eq
if t1 == t2 then
  print("PASS: __eq works")
else
  print("FAIL: __eq ignored")
end

print("Testing __lt...")
local lt_mt = {
  __lt = function(a, b)
    return true
  end
}
local t3 = setmetatable({}, lt_mt)
local t4 = setmetatable({}, lt_mt)
-- Should be true because of __lt
local status, res = pcall(function() return t3 < t4 end)
if status and res then
  print("PASS: __lt works")
else
  print("FAIL: __lt failed: " .. tostring(res))
end

print("Testing __le...")
local le_mt = {
  __le = function(a, b)
    return true
  end
}
local t5 = setmetatable({}, le_mt)
local t6 = setmetatable({}, le_mt)
-- Should be true because of __le
local status, res = pcall(function() return t5 <= t6 end)
if status and res then
  print("PASS: __le works")
else
  print("FAIL: __le failed: " .. tostring(res))
end

print("Testing __concat...")
local concat_mt = {
  __concat = function(a, b)
    return "concatenated"
  end
}
local t7 = setmetatable({}, concat_mt)
local t8 = setmetatable({}, concat_mt)
-- Should be "concatenated"
local status, res = pcall(function() return t7 .. t8 end)
if status and res == "concatenated" then
  print("PASS: __concat works")
else
  print("FAIL: __concat failed: " .. tostring(res))
end
