-- Test: math.max/math.min work with strings and __lt metamethods

-- String comparison (lexicographic)
assert(math.max("b", "a", "c") == "c")
assert(math.min("b", "a", "c") == "a")
assert(math.max("apple", "banana") == "banana")
assert(math.min("apple", "banana") == "apple")

-- Still works with numbers
assert(math.max(1, 3, 2) == 3)
assert(math.min(1, 3, 2) == 1)

-- Works with __lt metamethods
local mt = {__lt = function(a, b) return a.val < b.val end}
local a = setmetatable({val = 10}, mt)
local b = setmetatable({val = 20}, mt)
local c = setmetatable({val = 5}, mt)
assert(math.max(a, b, c) == b)
assert(math.min(a, b, c) == c)

print("OK")
