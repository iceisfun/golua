-- test_string_gmatch_pattern.lua
-- string.gmatch should iterate over Lua pattern matches and expose captures.

local matches = {}
for token in string.gmatch("abc123def456", "%d+") do
    matches[#matches + 1] = token
end
assert(#matches == 2, string.format("expected 2 digit chunks, got %d", #matches))
assert(matches[1] == "123" and matches[2] == "456", string.format("unexpected tokens: %s", table.concat(matches, ",")))

local assignments = {}
for name, value in string.gmatch("foo=1 bar=20 baz=300", "(%a+)=(%d+)") do
    assignments[name] = value
end
assert(assignments.foo == "1", "missing capture 'foo'")
assert(assignments.bar == "20", "missing capture 'bar'")
assert(assignments.baz == "300", "missing capture 'baz'")
