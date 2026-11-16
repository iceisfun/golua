-- Test: read('n') accepts a numeral of exactly 200 characters (L_MAXLENNUM)
-- and only fails when a 201st character would have to be stored.
local name = os.tmpname()
local function try(n)
  local f = io.open(name, "wb") f:write(string.rep('1', n), ' X') f:close()
  f = io.open(name, "rb")
  local num, rest = f:read('n'), f:read('a')
  f:close()
  return num, rest
end

local num, rest = try(199)
assert(num == tonumber(string.rep('1', 199)) and rest == " X")
num, rest = try(200)
assert(num == tonumber(string.rep('1', 200)) and rest == " X")
num, rest = try(201)
assert(num == nil and rest == "1 X")

os.remove(name)

-- the same limit applies to the reader io.popen handles use
local function trypopen(n)
  local f = assert(io.popen("printf '%0" .. n .. "d X' 1"))
  local num, rest = f:read('n'), f:read('a')
  f:close()
  return num, rest
end
num, rest = trypopen(199)
assert(num == 1 and rest == " X")
num, rest = trypopen(200)
assert(num == 1 and rest == " X")
num, rest = trypopen(201)
assert(num == nil and rest == "1 X")
print("PASS")
