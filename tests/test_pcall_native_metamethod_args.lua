-- Test: native functions called via pcall preserve args across metamethod calls
-- Regression test for register clobbering when ProtectedCall didn't advance vm.top

local pass, fail = 0, 0
local function check(name, got, expect)
  if got == expect then
    pass = pass + 1
  else
    fail = fail + 1
    print("FAIL " .. name .. ": got " .. tostring(got) .. ", expected " .. tostring(expect))
  end
end

local mt = { __lt = function(a, b) return a.v < b.v end }
local function obj(v) return setmetatable({v = v}, mt) end

-- math.max with 3+ metamethod args via pcall
do
  local ok, r = pcall(math.max, obj(1), obj(5), obj(3))
  check("pcall max 3 args ok", ok, true)
  check("pcall max 3 args val", r and r.v, 5)
end

-- math.min with 3+ metamethod args via pcall
do
  local ok, r = pcall(math.min, obj(7), obj(2), obj(9))
  check("pcall min 3 args ok", ok, true)
  check("pcall min 3 args val", r and r.v, 2)
end

-- math.max with 4 args via pcall
do
  local ok, r = pcall(math.max, obj(3), obj(1), obj(4), obj(2))
  check("pcall max 4 args ok", ok, true)
  check("pcall max 4 args val", r and r.v, 4)
end

-- math.min with 4 args via pcall
do
  local ok, r = pcall(math.min, obj(3), obj(1), obj(4), obj(2))
  check("pcall min 4 args ok", ok, true)
  check("pcall min 4 args val", r and r.v, 1)
end

-- Direct call (should also work, was already fine)
do
  local r = math.max(obj(1), obj(5), obj(3))
  check("direct max 3 args", r.v, 5)
end

-- xpcall variant
do
  local ok, r = xpcall(math.max, debug.traceback, obj(10), obj(20), obj(15))
  check("xpcall max 3 args ok", ok, true)
  check("xpcall max 3 args val", r and r.v, 20)
end

print(string.format("\npcall native metamethod args: %d passed, %d failed", pass, fail))
assert(fail == 0, fail .. " tests failed")
