-- Test: table.concat
-- From: strings.lua
-- What: Tests table.concat with various separators, ranges, null bytes, empty tables, maxinteger/mininteger indices, and error cases.

do
local maxi = math.maxinteger
local mini = math.mininteger

local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

checkerror("table expected", table.concat, 3)
checkerror("at index " .. maxi, table.concat, {}, " ", maxi, maxi)
checkerror("at index %" .. mini, table.concat, {}, " ", mini, mini)
assert(table.concat{} == "")
assert(table.concat({}, 'x') == "")
assert(table.concat({'\0', '\0\1', '\0\1\2'}, '.\0.') == "\0.\0.\0\1.\0.\0\1\2")
local a = {}; for i=1,300 do a[i] = "xuxu" end
assert(table.concat(a, "123").."123" == string.rep("xuxu123", 300))
assert(table.concat(a, "b", 20, 20) == "xuxu")
assert(table.concat(a, "", 20, 21) == "xuxuxuxu")
assert(table.concat(a, "x", 22, 21) == "")
assert(table.concat(a, "3", 299) == "xuxu3xuxu")
assert(table.concat({}, "x", maxi, maxi - 1) == "")
assert(table.concat({}, "x", mini + 1, mini) == "")
assert(table.concat({}, "x", maxi, mini) == "")
assert(table.concat({[maxi] = "alo"}, "x", maxi, maxi) == "alo")
assert(table.concat({[maxi] = "alo", [maxi - 1] = "y"}, "-", maxi - 1, maxi)
       == "y-alo")

assert(not pcall(table.concat, {"a", "b", {}}))

a = {"a","b","c"}
assert(table.concat(a, ",", 1, 0) == "")
assert(table.concat(a, ",", 1, 1) == "a")
assert(table.concat(a, ",", 1, 2) == "a,b")
assert(table.concat(a, ",", 2) == "b,c")
assert(table.concat(a, ",", 3) == "c")
assert(table.concat(a, ",", 4) == "")
end
