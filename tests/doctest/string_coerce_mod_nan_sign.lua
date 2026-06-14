-- Float modulo by zero yields NaN. C's fmod() returns a NEGATIVE NaN, which
-- prints as "-nan"; Go's math.Mod returns a positive NaN ("nan"). The VM's
-- luaNumMod already copysign-corrects this, but the string library's
-- arithmetic metamethod (the path taken when an operand is a coerced string)
-- duplicated the modulo logic WITHOUT the correction, so string-coerced
-- `% 0.0` printed "nan" instead of "-nan". Verify parity with reference Lua.

-- String-coerced operand goes through the string __mod metamethod.
print("3.5" % 0.0)
--> =-nan
print("10" % 0.0)
--> =-nan

-- Direct float path (for contrast) — already correct.
print(3.5 % 0.0)
--> =-nan
