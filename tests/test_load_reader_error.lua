-- load() reader error: returns raw error, not prefixed
local f, err = load(function() error("reader boom") end)
assert(f == nil)
assert(err:find("reader boom"), "expected raw error in: " .. tostring(err))
assert(not err:find("error calling"), "should not have 'error calling' prefix in: " .. err)

-- load() reader error with table: "(error object is a TYPE value)"
local f2, err2 = load(function() error({code=42}) end)
assert(f2 == nil)
assert(type(err2) == "string", "table error should be converted to string")
assert(err2:find("error object is a table value"), "expected table error format in: " .. err2)

-- load() reader returning non-string: should have file:line prefix
local f3, err3 = load(function() return true end)
assert(f3 == nil)
assert(err3:find("reader function must return a string"), "expected reader error in: " .. err3)
-- Should have file:line prefix
assert(err3:find(":%d+:"), "expected file:line prefix in: " .. err3)
