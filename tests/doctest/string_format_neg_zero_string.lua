-- string.format coerces a string argument to a number integer-first (like Lua's
-- luaO_str2num). The string "-0" is an INTEGER literal -> integer 0 -> +0.0, so a
-- float conversion prints "0", NOT "-0". A real float lexeme ("-0.0") or an actual
-- float -0.0 keeps the negative sign. Matches reference Lua 5.5.

-- integer-magnitude "-0"/"-00" lose the sign (parsed as integer 0)
print(string.format("%g", "-0"))
--> =0
print(string.format("%g", "-00"))
--> =0
print(string.format("%+g", "-0"))
--> =+0
print(string.format("%a", "-0"))
--> =0x0p+0

-- a real float lexeme keeps -0.0
print(string.format("%g", "-0.0"))
--> =-0
-- an actual float -0.0 keeps the sign
print(string.format("%g", -0.0))
--> =-0
-- ordinary numeric strings are unaffected
print(string.format("%g", "3.5"), string.format("%g", "5"))
--> =3.5	5
