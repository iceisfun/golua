-- Regression: `//` (integer division) traceback labels the [C] frame
-- `[C]: in metamethod 'idiv'` (matching lua5.5.0), not `'add'`.
-- Previously `decodeBytecodeMetamethodTag` subtracted 6 from any tag >= 6,
-- which aliased our TM_IDIV (ordinal 6) to TM_ADD (ordinal 0).

local function trigger() local x = "hello"; return x // 1 end

local ok, err = xpcall(trigger, debug.traceback)
assert(not ok, "expected error from string // number")
assert(err:find("attempt to idiv", 1, true),
  "expected 'attempt to idiv' in message, got: " .. tostring(err))
assert(err:find("metamethod 'idiv'", 1, true),
  "expected \"metamethod 'idiv'\" in traceback, got: " .. tostring(err))
assert(not err:find("metamethod 'add'", 1, true),
  "should NOT label idiv frame as metamethod 'add', got: " .. tostring(err))

-- Spot-check that other arithmetic / bitwise ops still label correctly,
-- driving them through the string metatable (arithmetic) and a table with
-- non-callable metamethod (bitwise) so a metamethod frame is produced.
local function check_str(op_fn, want_name)
  local ok, err = xpcall(op_fn, debug.traceback)
  assert(not ok, "expected error for " .. want_name)
  assert(err:find("metamethod '" .. want_name .. "'", 1, true),
    "expected metamethod '" .. want_name .. "' for " .. want_name ..
    ", got: " .. tostring(err))
end

check_str(function() local x = "hello"; return x + 1 end, "add")
check_str(function() local x = "hello"; return x - 1 end, "sub")
check_str(function() local x = "hello"; return x * 1 end, "mul")
check_str(function() local x = "hello"; return x / 1 end, "div")
check_str(function() local x = "hello"; return x % 1 end, "mod")
check_str(function() local x = "hello"; return x ^ 1 end, "pow")
check_str(function() local x = "hello"; return x // 1 end, "idiv")

-- Bitwise: string metatable doesn't provide bitwise, so set up a table with
-- a non-callable metamethod to get a "metamethod '<op>'" label.
local function check_bit(op_fn, want_name)
  local ok, err = xpcall(op_fn, debug.traceback)
  assert(not ok, "expected error for " .. want_name)
  assert(err:find("metamethod '" .. want_name .. "'", 1, true),
    "expected metamethod '" .. want_name .. "' for " .. want_name ..
    ", got: " .. tostring(err))
end

check_bit(function() local t = setmetatable({}, {__band = 1}); return t & 1 end, "band")
check_bit(function() local t = setmetatable({}, {__bor  = 1}); return t | 1 end, "bor")
check_bit(function() local t = setmetatable({}, {__bxor = 1}); return t ~ 1 end, "bxor")
check_bit(function() local t = setmetatable({}, {__shl  = 1}); return t << 1 end, "shl")
check_bit(function() local t = setmetatable({}, {__shr  = 1}); return t >> 1 end, "shr")

print("idiv metamethod traceback name ok")
