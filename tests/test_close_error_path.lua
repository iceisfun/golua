-- Bug 1: __close handler is called TWICE on error path (should be once).
-- Bug 2: __close handler receives nil as error argument (should receive the error value).

-- Test: __close called exactly once on error
local count = 0
local ok, e = pcall(function()
  local y<close> = setmetatable({}, {__close = function(self, err)
    count = count + 1
  end})
  error("boom")
end)
assert(not ok, "pcall should catch error")
assert(count == 1, "expected __close called 1 time, got: " .. count)

-- Test: __close receives the error value as second argument
local received_err = "not set"
ok, e = pcall(function()
  local y<close> = setmetatable({}, {__close = function(self, err)
    received_err = err
  end})
  error("boom")
end)
assert(not ok, "pcall should catch error")
assert(received_err ~= nil,
  "expected __close to receive error value, got nil")
assert(type(received_err) == "string",
  "expected __close error arg to be string, got: " .. type(received_err))
assert(string.find(received_err, "boom"),
  "expected __close error to contain 'boom', got: " .. tostring(received_err))

-- Test: __close on normal exit receives nil as error (should still work)
local normal_err = "not set"
do
  local y<close> = setmetatable({}, {__close = function(self, err)
    normal_err = err
  end})
end
assert(normal_err == nil,
  "expected nil error on normal close, got: " .. tostring(normal_err))

-- Test: multiple __close variables, each called exactly once, innermost first
local order = {}
ok = pcall(function()
  local a<close> = setmetatable({name="a"}, {__close = function(self, err)
    order[#order+1] = self.name
  end})
  local b<close> = setmetatable({name="b"}, {__close = function(self, err)
    order[#order+1] = self.name
  end})
  error("boom")
end)
assert(#order == 2, "expected 2 __close calls, got: " .. #order)
assert(order[1] == "b", "expected innermost (b) closed first, got: " .. tostring(order[1]))
assert(order[2] == "a", "expected outermost (a) closed second, got: " .. tostring(order[2]))

print("PASS")
