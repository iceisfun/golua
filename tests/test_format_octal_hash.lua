-- string.format %#o with value 0
-- Bug: %#.0o with value 0 produced empty string instead of "0"

assert(string.format("%#o", 0) == "0")
assert(string.format("%#.0o", 0) == "0")
assert(string.format("<%#.0o>", 0) == "<0>")
assert(string.format("%#10.0o", 0) == "         0")
assert(string.format("%-#10.0o", 0) == "0         ")

-- Non-zero values should still work
assert(string.format("%#o", 8) == "010")
assert(string.format("%#.0o", 8) == "010")

print("PASS")
