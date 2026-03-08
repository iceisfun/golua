-- To-be-closed variables in outer if-block should not be closed
-- when an inner nested block (do/for/if/repeat) exits.

-- Test 1: do-end inside if
local log = {}
if true then
  local b <close> = setmetatable({}, {__close = function() log[#log+1] = "b" end})
  do
    local c <close> = setmetatable({}, {__close = function() log[#log+1] = "c" end})
  end
  log[#log+1] = "mid"
end
local result = table.concat(log, ",")
assert(result == "c,mid,b", "test1: expected 'c,mid,b' got '" .. result .. "'")

-- Test 2: for-loop inside if
local log2 = {}
if true then
  local b <close> = setmetatable({}, {__close = function() log2[#log2+1] = "b" end})
  for i = 1, 1 do
    local c <close> = setmetatable({}, {__close = function() log2[#log2+1] = "c" end})
  end
  log2[#log2+1] = "mid"
end
local result2 = table.concat(log2, ",")
assert(result2 == "c,mid,b", "test2: expected 'c,mid,b' got '" .. result2 .. "'")

-- Test 3: nested if inside if
local log3 = {}
if true then
  local b <close> = setmetatable({}, {__close = function() log3[#log3+1] = "b" end})
  if true then
    local c <close> = setmetatable({}, {__close = function() log3[#log3+1] = "c" end})
  end
  log3[#log3+1] = "mid"
end
local result3 = table.concat(log3, ",")
assert(result3 == "c,mid,b", "test3: expected 'c,mid,b' got '" .. result3 .. "'")

-- Test 4: repeat inside if
local log4 = {}
if true then
  local b <close> = setmetatable({}, {__close = function() log4[#log4+1] = "b" end})
  repeat
    local c <close> = setmetatable({}, {__close = function() log4[#log4+1] = "c" end})
  until true
  log4[#log4+1] = "mid"
end
local result4 = table.concat(log4, ",")
assert(result4 == "c,mid,b", "test4: expected 'c,mid,b' got '" .. result4 .. "'")

-- Test 5: do-end inside do-end (non-if outer block)
local log5 = {}
do
  local b <close> = setmetatable({}, {__close = function() log5[#log5+1] = "b" end})
  do
    local c <close> = setmetatable({}, {__close = function() log5[#log5+1] = "c" end})
  end
  log5[#log5+1] = "mid"
end
local result5 = table.concat(log5, ",")
assert(result5 == "c,mid,b", "test5: expected 'c,mid,b' got '" .. result5 .. "'")

print("OK")
