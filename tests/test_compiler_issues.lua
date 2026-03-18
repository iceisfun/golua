-- Test compiler/parser issue fixes

-- Issue 1: Duplicate adjacent label error reports wrong line number
-- When two identical labels are directly adjacent, lua5.4 reports the
-- second label's line (due to recursive labelstat processing).
local f, e = load("::L::\n::L::")
assert(e ~= nil, "expected error for duplicate label")
-- lua5.4 says "already defined on line 2" (the second label's line)
assert(string.find(e, "already defined on line 2"), "expected 'already defined on line 2', got: " .. e)

-- Non-adjacent labels should still report the first label's line
local f2, e2 = load("::L::\nlocal x = 1\n::L::")
assert(e2 ~= nil, "expected error for duplicate label")
assert(string.find(e2, "already defined on line 1"), "expected 'already defined on line 1', got: " .. e2)

-- Three adjacent labels: ::L:: on lines 1,2,3 — should report line 3 (innermost recursive)
local f3, e3 = load("::L::\n::L::\n::L::")
assert(e3 ~= nil, "expected error for triple duplicate label")
-- lua5.4 recursive processing: processes 3rd first (registers line 3),
-- then 2nd finds duplicate on line 3, errors "already defined on line 3"
assert(string.find(e3, "already defined on line 3"),
  "expected 'already defined on line 3', got: " .. e3)

-- Issue 2a: activelines should include for-loop header line
local f4 = load([[
local f = function()
  for i = 1, 10 do
    local x = i
  end
end
return debug.getinfo(f, "SL")
]])
local info4 = f4()
local lines4 = {}
for k in pairs(info4.activelines) do lines4[#lines4+1] = k end
table.sort(lines4)
local result4 = table.concat(lines4, ",")
-- Expected: 2,3,5 (for-header, body, end)
assert(info4.activelines[2], "for-loop header line 2 missing from activelines, got: " .. result4)
assert(info4.activelines[3], "for-loop body line 3 missing from activelines, got: " .. result4)
assert(info4.activelines[5], "function end line 5 missing from activelines, got: " .. result4)

-- Issue 2b: activelines should NOT include if-true condition line
local f5 = load([[
local f = function()
  if true then
    return 1
  end
end
return debug.getinfo(f, "SL")
]])
local info5 = f5()
local lines5 = {}
for k in pairs(info5.activelines) do lines5[#lines5+1] = k end
table.sort(lines5)
local result5 = table.concat(lines5, ",")
-- Expected: 3,5 (body return, end) — NOT including line 2 (the if-condition)
assert(not info5.activelines[2], "if-true condition line 2 should NOT be in activelines, got: " .. result5)
assert(info5.activelines[3], "return line 3 should be in activelines, got: " .. result5)

-- Issue 2c: activelines SHOULD include if-variable condition line
local f6 = load([[
local f = function()
  if x then
    return 1
  end
end
return debug.getinfo(f, "SL")
]])
local info6 = f6()
local lines6 = {}
for k in pairs(info6.activelines) do lines6[#lines6+1] = k end
table.sort(lines6)
local result6 = table.concat(lines6, ",")
-- Expected: 2,3,5 (condition, body return, end)
assert(info6.activelines[2], "if-variable condition line 2 should be in activelines, got: " .. result6)

-- Issue 2d: generic for-in loop header should be in activelines
local f7 = load([[
local f = function()
  for k, v in pairs({}) do
    local x = k
  end
end
return debug.getinfo(f, "SL")
]])
local info7 = f7()
local lines7 = {}
for k in pairs(info7.activelines) do lines7[#lines7+1] = k end
table.sort(lines7)
local result7 = table.concat(lines7, ",")
-- The for-in header line should be included
assert(info7.activelines[2], "for-in header line 2 missing from activelines, got: " .. result7)

-- Issue 3: Deep nesting error message should say "C stack overflow"
local code = "return " .. string.rep("(", 200) .. "1" .. string.rep(")", 200)
local f8, e8 = load(code)
assert(f8 == nil, "expected error for deep nesting")
assert(string.find(e8, "C stack overflow"), "expected 'C stack overflow', got: " .. e8)

print("PASS")
