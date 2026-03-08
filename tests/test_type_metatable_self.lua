-- Test OP_SELF with type metatables (numbers, booleans, functions, nil)

-- Number type metatable with __index
debug.setmetatable(0, {__index = {double = function(self) return self * 2 end}})
local n = 5
assert(n:double() == 10, "number:method() should work via type metatable")
assert(n.double(n) == 10, "number.method should also work")
debug.setmetatable(0, nil)

-- Boolean type metatable with __index
debug.setmetatable(true, {__index = {test = function(self) return type(self) end}})
local b = true
assert(b:test() == "boolean", "boolean:method() should work via type metatable")
debug.setmetatable(true, nil)

-- Function type metatable with __index
local f = function() return 42 end
debug.setmetatable(f, {__index = {info = function(self) return "func" end}})
assert(f:info() == "func", "function:method() should work via type metatable")
debug.setmetatable(f, nil)

-- Nil type metatable with __index
debug.setmetatable(nil, {__index = {test = function(self) return "nil" end}})
local x = nil
assert(x:test() == "nil", "nil:method() should work via type metatable")
debug.setmetatable(nil, nil)

-- __newindex on number type metatable
local captured = {}
debug.setmetatable(0, {__newindex = function(self, k, v) captured[k] = v end})
local m = 5
m.x = 10
m[1] = "hello"
assert(captured.x == 10, "__newindex on number should work for field")
assert(captured[1] == "hello", "__newindex on number should work for index")
debug.setmetatable(0, nil)

-- __newindex on boolean type metatable
captured = {}
debug.setmetatable(true, {__newindex = function(self, k, v) captured[k] = v end})
local bt = true
bt.y = 20
assert(captured.y == 20, "__newindex on boolean should work")
debug.setmetatable(true, nil)

-- __newindex on nil type metatable
captured = {}
debug.setmetatable(nil, {__newindex = function(self, k, v) captured[k] = v end})
local nt = nil
nt.z = 30
assert(captured.z == 30, "__newindex on nil should work")
debug.setmetatable(nil, nil)

-- __newindex on function type metatable
captured = {}
local fn = print
debug.setmetatable(fn, {__newindex = function(self, k, v) captured[k] = v end})
fn.w = 40
assert(captured.w == 40, "__newindex on function should work")
debug.setmetatable(fn, nil)

-- __index as table (not function)
debug.setmetatable(0, {__index = {squared = function(self) return self * self end}})
local q = 4
assert(q:squared() == 16, "__index table should work with OP_SELF")
debug.setmetatable(0, nil)

-- __newindex as table
local store = {}
debug.setmetatable(0, {__newindex = store})
local r = 7
r.val = 99
assert(store.val == 99, "__newindex table should work for number")
debug.setmetatable(0, nil)

print("OK")
