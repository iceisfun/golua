-- utf8.char must use luaL_checkinteger semantics: a coercible numeric string is
-- accepted, but a non-numeric value reports "number expected, got <type>"
-- (a type error), distinct from a float with no integer representation which
-- reports "number has no integer representation". Matches Lua 5.5.

-- coercible numeric string works
print(utf8.char("65"))
--> =A

-- non-numeric string is a type error, not "no integer representation"
print(pcall(utf8.char, "abc"))
--> =false	bad argument #1 to 'utf8.char' (number expected, got string)

print(pcall(utf8.char, ""))
--> =false	bad argument #1 to 'utf8.char' (number expected, got string)

-- float with no integer representation keeps its own message
print(pcall(utf8.char, 65.5))
--> =false	bad argument #1 to 'utf8.char' (number has no integer representation)

-- other non-number types
print(pcall(utf8.char, nil))
--> =false	bad argument #1 to 'utf8.char' (number expected, got nil)

print(pcall(utf8.char, true))
--> =false	bad argument #1 to 'utf8.char' (number expected, got boolean)

-- error reports the correct argument position
print(pcall(utf8.char, 65, "x"))
--> =false	bad argument #2 to 'utf8.char' (number expected, got string)
