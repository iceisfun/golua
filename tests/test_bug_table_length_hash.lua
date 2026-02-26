-- Bug: # operator returns wrong length for tables built with constructors
-- containing leading nils. Lua 5.4's table constructor pre-allocates the
-- array part, so {nil, "a"} has array size 2. GoLua's SetInt refuses to
-- extend the array with nil, so values end up in the hash part and # returns 0.

-- Constructor with leading nil
assert(#{nil, "a"} == 2, "#{nil, 'a'} should be 2, got " .. #{nil, "a"})

-- Constructor with multiple leading nils
assert(#{nil, nil, 3} == 3, "#{nil, nil, 3} should be 3, got " .. #{nil, nil, 3})

-- Constructor with interior nil
assert(#{1, nil, 3} == 3, "#{1, nil, 3} should be 3, got " .. #{1, nil, 3})

-- Guard: normal sequential tables should keep working
assert(#{1, 2, 3} == 3)
assert(#{} == 0)
assert(#{"a"} == 1)

-- table.concat should error on {nil, "a"} with default range
-- (because # returns 2, concat iterates 1..2 and hits nil at index 1)
local ok, err = pcall(table.concat, {nil, "a"}, ",")
assert(not ok, "table.concat({nil,'a'}) should error with default range")
