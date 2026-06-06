-- Integer divide-by-zero and n%0 errors raised from a STRING-coerced operand
-- go through the string library's arithmetic metamethod, which is a C frame in
-- reference Lua. Such errors therefore carry NO source:line: position prefix,
-- unlike the same operation on plain integer operands (which is folded/handled
-- in the VM and does carry a position).

-- String operand: position-less error (matches lua5.5.0)
print(pcall(function() return "010" // 0 end))
--> =false	attempt to divide by zero
print(pcall(function() return "010" % 0 end))
--> =false	attempt to perform 'n%0'

-- The error value is a plain string of exactly that text
local ok, msg = pcall(function() return "5" // 0 end)
print(ok, msg, type(msg))
--> =false	attempt to divide by zero	string

-- Non-numeric string still errors with a position (different code path)
print(pcall(function() return "x" // 0 end))
--> ~^false\t.*:\d+: attempt to idiv a 'string' with a 'number'
