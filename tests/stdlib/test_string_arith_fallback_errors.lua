-- Test: string arithmetic falling back to the other operand's metamethod
-- What: lstrlib's trymt calls the other operand's metamethod with a plain
-- lua_call, so an error it raises reaches the caller unchanged (any value,
-- no added position) and a yield inside it crosses a C-call boundary.

local mt = {}
mt.__sub = function(a, b) error(b.v, 0) end
local function o(v) return setmetatable({v = v}, mt) end

do
  local err = {code = 42}
  local ok, e = pcall(function() return "abc" - o(err) end)
  assert(not ok and e == err, "error table lost: " .. tostring(e))
end

do
  local ok, e = pcall(function() return "abc" - o("msg") end)
  assert(not ok and e == "msg", "error(msg, 0) gained a position: " .. tostring(e))

  ok, e = pcall(function() return "10" - o(7) end)
  assert(not ok and e == 7 and math.type(e) == "integer", tostring(e))

  ok, e = pcall(function()
    return "x" ^ setmetatable({}, {__pow = function() error(false) end})
  end)
  assert(not ok and e == false, tostring(e))
end

do
  local ok, e = pcall(function()
    return "x" + setmetatable({}, {__add = function() error("boom") end})
  end)
  assert(not ok, "expected error")
  assert(e:match("^[^:]+:%d+: boom$"), "position added twice: " .. e)
end

do
  local co = coroutine.create(function()
    return "x" * setmetatable({}, {__mul = function() return coroutine.yield(1) end})
  end)
  local ok, e = coroutine.resume(co)
  assert(not ok and e == "attempt to yield across a C-call boundary", tostring(e))
end

do
  -- The successful fallback still returns the metamethod's result.
  local r = "1" + setmetatable({}, {__add = function() return 99 end})
  assert(r == 99)
end
