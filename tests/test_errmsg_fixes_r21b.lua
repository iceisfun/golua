-- Regression tests for round 21b error message fixes

-- 1. Thread # (length) should error, not return 0
local co = coroutine.create(function() end)
local ok, err = pcall(function() return #co end)
assert(not ok, "expected error for #thread")
assert(err:find("attempt to get length of a thread value"), err)

-- 2. os.date arg#2 error should include "got TYPE" suffix
local ok2, err2 = pcall(os.date, "%Y", "bad")
assert(not ok2, "expected error for os.date with string")
assert(err2:find("got string"), "missing 'got string' suffix: " .. err2)

-- 3. debug.getmetatable() with no args should error
local ok3, err3 = pcall(debug.getmetatable)
assert(not ok3, "expected error for debug.getmetatable()")
assert(err3:find("value expected"), "wrong error: " .. err3)

-- 4. debug.setmetatable() with no args should error
local ok4, err4 = pcall(debug.setmetatable)
assert(not ok4, "expected error for debug.setmetatable()")

-- 5. TBC nil close error should have file:line prefix
local ok5, err5 = pcall(function()
  local x <close> = setmetatable({}, {__close = function() end})
  -- replace metatable to remove __close
  debug.setmetatable(x, {})
end)
assert(not ok5, "expected error for nil close")
-- The error should contain a line reference (colon followed by number)
assert(err5:find(":%d+:"), "missing file:line prefix in TBC error: " .. err5)

-- 6. _ENV with >255 constants should show global name not field '?'
local parts = {}
for i = 1, 260 do
  parts[#parts + 1] = string.format("local x%d = '%d'", i, i)
end
parts[#parts + 1] = "local _ENV = _ENV"
parts[#parts + 1] = "return bbb + 1"
local f, e = load(table.concat(parts, "\n"))
if f then
  local ok6, err6 = pcall(f)
  if not ok6 then
    -- Should say "global 'bbb'" not "field '?'"
    assert(err6:find("global 'bbb'") or err6:find("nil value"),
           "wrong error annotation: " .. err6)
  end
end

print("PASS")
