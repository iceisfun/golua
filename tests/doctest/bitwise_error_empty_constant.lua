-- A bitwise operation on a string constant annotates the offending operand
-- with "(constant '<value>')". The empty string is a legitimate constant name,
-- so it must be annotated too — the bytecode name resolver signals "not found"
-- via an empty 'what', not an empty name. Matches reference Lua 5.5.

print(pcall(function() return 0 & "" end))
--> ~^false\t.*: attempt to perform bitwise operation on a string value \(constant ''\)$

print(pcall(function() return 0 & "abc" end))
--> ~^false\t.*: attempt to perform bitwise operation on a string value \(constant 'abc'\)$

print(pcall(function() return "" & 0 end))
--> ~^false\t.*: attempt to perform bitwise operation on a string value \(constant ''\)$

print(pcall(function() return ~"" end))
--> ~^false\t.*: attempt to perform bitwise operation on a string value \(constant ''\)$
