-- Lua 5.5: for-loop control variables are read-only (compile-time error)

-- Numeric for: assigning to i is an error
local f, err = load("for i = 1, 10 do i = 5 end")
print(f == nil)
--> =true
print(err)
--> ~attempt to assign to const variable 'i'

-- Generic for: assigning to first variable (control) is an error
local f2, err2 = load("for k, v in pairs({}) do k = 1 end")
print(f2 == nil)
--> =true
print(err2)
--> ~attempt to assign to const variable 'k'

-- Generic for: assigning to second variable is OK (not the control variable)
local f3, err3 = load("for k, v in pairs({}) do v = 1 end")
print(f3 ~= nil)
--> =true
print(err3)
--> =nil

-- Normal read usage is fine
local sum = 0
for i = 1, 3 do
  sum = sum + i
end
print(sum)
--> =6

-- Shadowing with local is fine
local results = {}
for i = 1, 3 do
  local i = i + 10
  results[#results + 1] = i
end
print(table.concat(results, ","))
--> =11,12,13

-- Generic for normal usage is fine
local keys = {}
for k, v in pairs({a=1, b=2}) do
  keys[#keys + 1] = k
end
table.sort(keys)
print(table.concat(keys, ","))
--> =a,b
