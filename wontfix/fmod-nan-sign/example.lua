-- Sign of NaN from math.fmod / the % operator.
--
-- When a finite dividend meets an explicitly negative NaN divisor, golua and
-- reference Lua can print the NaN with a different sign ("-nan" vs "nan").
-- The numeric result is NaN either way; only the sign bit (which has no
-- arithmetic meaning for NaN) differs.

print(math.fmod(2, -(0/0)))
--> golua:    -nan
--> lua5.5.0:  nan

-- golua AGREES with the reference for every common NaN source — the divergence
-- only appears with an explicitly negated NaN operand:
print(0/0)                  --> both: -nan
print((0/0) % 1)            --> both: -nan
print(math.fmod(2, 0/0))    --> both: -nan
