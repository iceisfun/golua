-- When __tostring errors, the error should propagate with a single
-- file:line prefix, not a double prefix. Previously golua added the
-- tostring() call site prefix on top of the inner error's prefix.

local ok, err = pcall(function()
    return tostring(setmetatable({}, {
        __tostring = function() return string.format("%d", {}) end
    }))
end)

-- Should have exactly one file:line prefix, not two
-- Count occurrences of the filename pattern
local count = 0
for _ in tostring(err):gmatch(":%d+:") do count = count + 1 end
print(count)
--> =1
