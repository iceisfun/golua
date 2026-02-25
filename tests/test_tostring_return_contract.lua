-- test_tostring_return_contract: __tostring return-type behavior should match Lua 5.4

-- Number return is allowed and coerced to string
local n = setmetatable({}, { __tostring = function() return 42 end })
local rn = tostring(n)
assert(type(rn) == "string" and rn == "42", "numeric __tostring result should coerce to string")

-- Non-string/non-number return must error
local bad_table = setmetatable({}, { __tostring = function() return {} end })
local ok1, err1 = pcall(function() return tostring(bad_table) end)
assert(ok1 == false, "__tostring returning table should error")
assert(type(err1) == "string" and err1:find("__tostring") and err1:find("must return a string"),
       "unexpected error for table return: " .. tostring(err1))

local bad_bool = setmetatable({}, { __tostring = function() return true end })
local ok2, err2 = pcall(function() return tostring(bad_bool) end)
assert(ok2 == false, "__tostring returning boolean should error")
assert(type(err2) == "string" and err2:find("__tostring") and err2:find("must return a string"),
       "unexpected error for bool return: " .. tostring(err2))

local bad_nil = setmetatable({}, { __tostring = function() return nil end })
local ok3, err3 = pcall(function() return tostring(bad_nil) end)
assert(ok3 == false, "__tostring returning nil should error")
assert(type(err3) == "string" and err3:find("__tostring") and err3:find("must return a string"),
       "unexpected error for nil return: " .. tostring(err3))
