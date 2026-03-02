-- Test: Method with vararg and self
-- From: vararg.lua
-- What: Tests that a method using ... with self correctly indexes into self.

do
local t = {1, 10}
function t:f (...) local arg = {...}; return self[...]+#arg end
assert(t:f(1,4) == 3 and t:f(2) == 11)
end
