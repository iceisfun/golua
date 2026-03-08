-- Test: load() reader error should not include traceback when called via pcall
local function reader() error("boom") end

-- Via pcall, the error message should be clean (no traceback)
local ok, f, err = pcall(load, reader)
assert(ok == true, "pcall should succeed")
assert(f == nil, "load should return nil on reader error")
assert(type(err) == "string", "error should be a string, got " .. type(err))
assert(not string.find(err, "stack traceback"),
    "error should not contain stack traceback, got: " .. tostring(err))
assert(string.find(err, "boom"), "error should contain 'boom', got: " .. tostring(err))

print("OK")
