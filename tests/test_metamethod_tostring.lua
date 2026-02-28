-- Test: tostring() must honor __tostring metamethod
-- Covers: basic dispatch, self access, non-string returns, concat
-- integration, nested tostring, and absence of __tostring.

-- 1. Basic __tostring dispatch
local obj = setmetatable({}, {
    __tostring = function() return "custom_str" end
})
assert(tostring(obj) == "custom_str",
    "basic __tostring failed: " .. tostring(obj))

-- 2. __tostring receives the table as argument
local named = setmetatable({name = "alice"}, {
    __tostring = function(self) return "User:" .. self.name end
})
assert(tostring(named) == "User:alice",
    "__tostring self access failed: " .. tostring(named))

-- 3. __tostring returning a number (should still work, tostring coerces)
local numobj = setmetatable({}, {
    __tostring = function() return 42 end
})
local r = tostring(numobj)
-- Lua 5.4: tostring returns whatever __tostring returns
assert(r == 42 or r == "42", "__tostring returning number failed")

-- 4. Table without __tostring falls back to default "table: 0x..."
local plain = {}
local s = tostring(plain)
assert(type(s) == "string" and s:sub(1, 6) == "table:",
    "plain table tostring should start with 'table:', got: " .. s)

-- 5. Table with metatable but no __tostring falls back to default
local mt_no_ts = setmetatable({}, { __index = function() return 1 end })
local s2 = tostring(mt_no_ts)
assert(type(s2) == "string" and s2:sub(1, 6) == "table:",
    "metatable without __tostring should fall back, got: " .. s2)

-- 6. __tostring works inside string concatenation via tostring()
local tag = setmetatable({val = "X"}, {
    __tostring = function(self) return "[" .. self.val .. "]" end
})
assert("prefix-" .. tostring(tag) .. "-suffix" == "prefix-[X]-suffix",
    "__tostring in concat failed")

-- 7. __tostring works with pcall (no errors leak)
local ok, val = pcall(tostring, obj)
assert(ok and val == "custom_str",
    "__tostring via pcall failed")

-- 8. __tostring that errors is propagated
local bad = setmetatable({}, {
    __tostring = function() error("tostring_boom", 0) end
})
local ok2, err2 = pcall(tostring, bad)
assert(not ok2, "__tostring error should propagate")
assert(type(err2) == "string" and err2:find("tostring_boom"),
    "__tostring error message wrong: " .. tostring(err2))

-- 9. Nested tostring: __tostring calls tostring on another meta-object
local inner = setmetatable({}, {
    __tostring = function() return "INNER" end
})
local outer = setmetatable({child = inner}, {
    __tostring = function(self) return "outer(" .. tostring(self.child) .. ")" end
})
assert(tostring(outer) == "outer(INNER)",
    "nested __tostring failed: " .. tostring(outer))
