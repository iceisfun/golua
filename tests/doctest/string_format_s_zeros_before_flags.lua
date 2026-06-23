-- For the %s conversion with modifiers, Lua checks the "string contains zeros"
-- condition BEFORE validating the flag/conversion combination. So a malformed
-- spec like "% s" applied to a zero-containing argument reports
-- "string contains zeros", not "invalid conversion specification". Matches 5.5.

-- zeros check fires before the (invalid) flag check
print(pcall(string.format, "% s", "\0\0"))
--> =false	bad argument #2 to 'string.format' (string contains zeros)

print(pcall(string.format, "%+.3s", "a\0b"))
--> =false	bad argument #2 to 'string.format' (string contains zeros)

-- a valid %s spec still reports zeros for a zero-containing argument
print(pcall(string.format, "%.3s", "a\0b"))
--> =false	bad argument #2 to 'string.format' (string contains zeros)

-- but with a zero-free argument the invalid-flag check still fires
print(pcall(string.format, "% s", "ab"))
--> =false	invalid conversion specification: '% s'

print(pcall(string.format, "%+s", "ab"))
--> =false	invalid conversion specification: '%+s'

print(pcall(string.format, "%05s", "ab"))
--> =false	invalid conversion specification: '%05s'

-- valid flags/width still work
print(string.format("%-5s|", "ab"))
--> =ab   |

-- bare %s with embedded zeros (no modifiers) is allowed
print((string.format("%s", "a\0b") == "a\0b"))
--> =true

-- Oversized width/precision is checked AFTER the zeros check for %s: a
-- zero-containing argument reports "string contains zeros" even when the width
-- is also too large; a zero-free argument reports the spec error. Matches 5.5.
print(pcall(string.format, "%100s", "a\0b"))
--> =false	bad argument #2 to 'string.format' (string contains zeros)

print(pcall(string.format, "%100s", "ab"))
--> =false	invalid conversion specification: '%100s'

-- An invalid conversion CHARACTER (e.g. 'F', unsupported by Lua) is rejected
-- with "invalid conversion '<spec>' to 'format'" BEFORE the width/precision
-- "invalid conversion specification" error — so '%123F' reports the former
-- while '%100d' (valid char, oversized width) reports the latter. Matches 5.5.
print(pcall(string.format, "%123F", 1.0))
--> =false	invalid conversion '%123F' to 'format'

print(pcall(string.format, "%100d", 1))
--> =false	invalid conversion specification: '%100d'
