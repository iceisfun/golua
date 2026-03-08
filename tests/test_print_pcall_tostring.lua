-- Bug: print() drops arguments after __tostring metamethod when called via pcall
-- Direct print(a, b) works, but pcall(print, a, b) loses 'b'

local mt = {__tostring = function(self) return tostring(self.val) end}
local a = setmetatable({val = 1}, mt)
local b = setmetatable({val = 2}, mt)
local c = setmetatable({val = 3}, mt)

-- Direct call works fine
print(a, b, c)  --> 1	2	3

-- Through pcall, arguments should not get lost
pcall(print, a, b, c)  --> 1	2	3
