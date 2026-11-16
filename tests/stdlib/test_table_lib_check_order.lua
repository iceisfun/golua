-- Test: table.unpack / table.concat / table.move follow reference check order
-- What: lua 5.5 ltablib checks the table and takes its length (running __len)
-- before reading the optional arguments, strings pass checktab without __len,
-- and table.move compares tables with __eq to pick its copy direction

local function lenerr()
  return setmetatable({}, {__len = function() error("lenerr", 0) end})
end

-- unpack always calls __len, even when j is given
local calls = 0
local t = setmetatable({10, 20, 30}, {__len = function() calls = calls + 1; return 3 end})
local a, b = table.unpack(t, 1, 2)
assert(a == 10 and b == 20 and calls == 1)

local ok, err = pcall(table.unpack, setmetatable({}, {__len = function() return 2.5 end}), 1, 2)
assert(not ok and err:find("object length is not an integer", 1, true), err)

ok, err = pcall(table.unpack, nil, 1, 2)
assert(not ok and err:find("table expected, got nil", 1, true), err)
ok, err = pcall(table.unpack, 42, 1, 1)
assert(not ok and err:find("table expected, got number", 1, true), err)
ok, err = pcall(table.unpack)
assert(not ok and err:find("table expected, got no value", 1, true), err)

-- __len error wins over a bad i/j
ok, err = pcall(table.unpack, lenerr(), "x")
assert(not ok and err == "lenerr", err)

-- concat: __len runs before sep/i/j are validated
ok, err = pcall(table.concat, lenerr(), {})
assert(not ok and err == "lenerr", err)
ok, err = pcall(table.concat, lenerr(), ",", "x")
assert(not ok and err == "lenerr", err)
calls = 0
t = setmetatable({}, {__len = function() calls = calls + 1; return 0 end})
ok, err = pcall(table.concat, t, {})
assert(not ok and err:find("bad argument #2", 1, true) and calls == 1, err)

-- strings pass checktab without a __len metamethod
assert(table.concat("") == "")
ok, err = pcall(table.concat, "abc")
assert(not ok and err:find("invalid value (nil) at index 1", 1, true), err)
do
  local mt = getmetatable("")
  local old = mt.__index
  mt.__index = function(s, i)
    if type(i) == "number" then return s:sub(i, i) end
    return string[i]
  end
  local ok2, res = pcall(table.concat, "hello", ",")
  mt.__index = old
  assert(ok2 and res == "h,e,l,l,o", res)
end

-- move: tables that __eq calls equal are copied as the same table
local eq = function() return true end
local log = {}
local ta = setmetatable({1, 2, 3, 4, 5}, {__eq = eq})
local tb = setmetatable({}, {__eq = eq, __newindex = function(t, k, v)
  log[#log + 1] = k
  rawset(t, k, v)
end})
table.move(ta, 1, 3, 2, tb)
assert(table.concat(log, ",") == "4,3,2", table.concat(log, ","))

-- the proxy case that corrupts data when copied forwards
local store = {1, 2, 3, 4, 5}
local pmt = {
  __index = function(_, k) return store[k] end,
  __newindex = function(_, k, v) store[k] = v end,
  __eq = eq,
}
table.move(setmetatable({}, pmt), 1, 3, 2, setmetatable({}, pmt))
assert(table.concat(store, ",") == "1,1,2,3,5", table.concat(store, ","))

-- distinct, unequal tables still copy forwards
log = {}
local tc = setmetatable({}, {__newindex = function(t, k, v)
  log[#log + 1] = k
  rawset(t, k, v)
end})
table.move({1, 2, 3, 4, 5}, 1, 3, 2, tc)
assert(table.concat(log, ",") == "2,3,4")
