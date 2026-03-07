-- coroutine.wrap adds caller location to string errors
local w = coroutine.wrap(function()
    coroutine.yield()
    error("boom")
end)
w()

local function caller()
    return w()
end
local ok, err = pcall(caller)
assert(not ok)
-- Error should have TWO file:line prefixes: caller's and error's
-- The caller's prefix is the one added by wrap
local count = 0
for _ in err:gmatch(":[%d]+:") do count = count + 1 end
assert(count >= 2, "expected >=2 file:line prefixes, got " .. count .. " in: " .. err)

-- Table errors should NOT get prefix
local w2 = coroutine.wrap(function() error({code = 42}) end)
local ok2, err2 = pcall(w2)
assert(not ok2)
assert(type(err2) == "table" and err2.code == 42, "table error preserved")

print("OK")
