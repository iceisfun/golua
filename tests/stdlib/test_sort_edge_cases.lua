-- Test: table.sort edge cases (empty, equal, strings)
-- From: sort.lua
-- What: Tests table.sort on empty arrays, arrays of all-equal values (using function(x,y) return nil end), and arrays of strings including those with embedded null/high bytes.

do
local function check (a, f)
  f = f or function (x,y) return x<y end;
  for n = #a, 2, -1 do
    assert(not f(a[n], a[n-1]))
  end
end

table.sort{}  -- empty array

local limit = 50000
local a = {}
for i=1,limit do a[i] = false end
table.sort(a, function(x,y) return nil end)

for i,v in pairs(a) do assert(v == false) end

AA = {"\xE1lo", "\0first :-)", "alo", "then this one", "45", "and a new"}
table.sort(AA)
check(AA)
end
