-- Test: stdlib functions called via pcall preserve args across metamethod calls
-- Covers table.concat, table.unpack, math.max, math.min with __index/__len/__lt

local pass, fail = 0, 0
local function check(name, got, expect)
  if got == expect then
    pass = pass + 1
  else
    fail = fail + 1
    print("FAIL " .. name .. ": got " .. tostring(got) .. ", expected " .. tostring(expect))
  end
end

----------------------------------------------------------------------
-- table.concat via pcall with __index proxy
----------------------------------------------------------------------
do
  local data = {"hello", "world", "test"}
  local proxy = setmetatable({}, {
    __index = function(_, k) return data[k] end,
    __len = function() return #data end
  })

  local ok, result = pcall(table.concat, proxy, "-")
  check("concat proxy ok", ok, true)
  check("concat proxy result", result, "hello-world-test")

  -- With explicit i, j args (tests that args 3,4 survive __len/__index)
  local ok2, result2 = pcall(table.concat, proxy, "+", 2, 3)
  check("concat proxy i,j ok", ok2, true)
  check("concat proxy i,j result", result2, "world+test")
end

----------------------------------------------------------------------
-- table.unpack via pcall with __index proxy
----------------------------------------------------------------------
do
  local data = {10, 20, 30, 40}
  local proxy = setmetatable({}, {
    __index = function(_, k) return data[k] end,
    __len = function() return #data end
  })

  local ok, a, b, c, d = pcall(table.unpack, proxy)
  check("unpack proxy ok", ok, true)
  check("unpack proxy a", a, 10)
  check("unpack proxy b", b, 20)
  check("unpack proxy c", c, 30)
  check("unpack proxy d", d, 40)

  -- With explicit i, j (tests args 2,3 survive __len)
  local ok2, x, y = pcall(table.unpack, proxy, 2, 3)
  check("unpack proxy i,j ok", ok2, true)
  check("unpack proxy x", x, 20)
  check("unpack proxy y", y, 30)
end

----------------------------------------------------------------------
-- math.max via pcall with __lt metamethod (3+ args)
----------------------------------------------------------------------
do
  local mt = { __lt = function(a, b) return a.v < b.v end }
  local function obj(v) return setmetatable({v = v}, mt) end

  -- 3 args
  local ok, r = pcall(math.max, obj(1), obj(5), obj(3))
  check("max 3 via pcall ok", ok, true)
  check("max 3 via pcall val", r and r.v, 5)

  -- 5 args
  local ok2, r2 = pcall(math.max, obj(2), obj(8), obj(1), obj(6), obj(4))
  check("max 5 via pcall ok", ok2, true)
  check("max 5 via pcall val", r2 and r2.v, 8)
end

----------------------------------------------------------------------
-- math.min via pcall with __lt metamethod (3+ args)
----------------------------------------------------------------------
do
  local mt = { __lt = function(a, b) return a.v < b.v end }
  local function obj(v) return setmetatable({v = v}, mt) end

  local ok, r = pcall(math.min, obj(7), obj(2), obj(9), obj(1))
  check("min 4 via pcall ok", ok, true)
  check("min 4 via pcall val", r and r.v, 1)
end

----------------------------------------------------------------------
-- xpcall variants (handler + metamethod interaction)
----------------------------------------------------------------------
do
  local mt = { __lt = function(a, b) return a.v < b.v end }
  local function obj(v) return setmetatable({v = v}, mt) end

  local ok, r = xpcall(math.max, debug.traceback, obj(10), obj(20), obj(15))
  check("xpcall max ok", ok, true)
  check("xpcall max val", r and r.v, 20)
end

----------------------------------------------------------------------
-- Nested pcall: pcall(pcall(math.max, ...)) with metamethods
----------------------------------------------------------------------
do
  local mt = { __lt = function(a, b) return a.v < b.v end }
  local function obj(v) return setmetatable({v = v}, mt) end

  local ok1, ok2, r = pcall(pcall, math.max, obj(3), obj(7), obj(1))
  check("nested pcall ok1", ok1, true)
  check("nested pcall ok2", ok2, true)
  check("nested pcall val", r and r.v, 7)
end

----------------------------------------------------------------------
-- Hook firing during pcall + metamethod doesn't clobber
----------------------------------------------------------------------
do
  local hook_count = 0
  local mt = { __lt = function(a, b) return a.v < b.v end }
  local function obj(v) return setmetatable({v = v}, mt) end

  -- Set a count hook that fires frequently
  debug.sethook(function() hook_count = hook_count + 1 end, "", 1)

  local ok, r = pcall(math.max, obj(1), obj(5), obj(3))
  debug.sethook()

  check("hook+pcall+meta ok", ok, true)
  check("hook+pcall+meta val", r and r.v, 5)
  check("hook fired", hook_count > 0, true)
end

----------------------------------------------------------------------
print(string.format("\npcall stdlib metamethod args: %d passed, %d failed", pass, fail))
assert(fail == 0, fail .. " tests failed")
