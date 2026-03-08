-- Test: String tokens in "near" clause should include their delimiter quotes
-- Bug: GoLua strips string delimiters in the "near" token of error messages.
-- Lua 5.4: near ''str''   (single-quoted string shown with quotes)
-- GoLua:   near 'str'     (just the content)

-- Single-quoted string
local _, err = load("local x = 1; 'str'")
assert(err:find("near ''str''"), "single-quoted: got: " .. tostring(err))

-- Double-quoted string
_, err = load('local x = 1; "str"')
assert(err:find('near \'"str"\''), "double-quoted: got: " .. tostring(err))

-- Long string [[...]]
_, err = load("local x = 1; [[str]]")
assert(err:find("near '%[%[str%]%]'"), "long string: got: " .. tostring(err))

-- Long string with level
_, err = load("local x = 1; [=[str]=]")
assert(err:find("near '%[=%[str%]=%]'"), "level1 string: got: " .. tostring(err))

-- Empty single-quoted string
-- Lua 5.4 shows: near '''' (outer quotes wrapping inner '')
_, err = load("local x = 1; ''")
assert(err:find("''''"), "empty string: got: " .. tostring(err))

print("PASS")
