-- A thread is backed by a table internally, but library functions that
-- require a table must reject it like any other non-table value.

local co = coroutine.create(print)

local function expect_error(want, f, ...)
  local ok, err = pcall(f, ...)
  assert(ok == false, "expected an error: " .. want)
  assert(string.find(tostring(err), want, 1, true) ~= nil, tostring(err))
end

expect_error("bad argument #2 to 'setmetatable' (nil or table expected, got thread)",
  setmetatable, {}, co)
expect_error("bad argument #2 to 'debug.setmetatable' (nil or table expected, got thread)",
  debug.setmetatable, {}, co)
expect_error("bad argument #3 to 'string.gsub' (string/function/table expected, got thread)",
  string.gsub, "abc", "%w", co)

-- The rejected calls must not have installed anything.
local t = {}
pcall(setmetatable, t, co)
assert(getmetatable(t) == nil)
pcall(debug.setmetatable, t, co)
assert(getmetatable(t) == nil)

-- os.time reads date fields from its table argument; a thread is not one.
expect_error("bad argument #1 to 'os.time' (table expected, got thread)", os.time, co)
debug.setmetatable(co, {__name = "MyThread", __index = function() return 1 end})
expect_error("bad argument #1 to 'os.time' (table expected, got MyThread)", os.time, co)
debug.setmetatable(co, nil)
