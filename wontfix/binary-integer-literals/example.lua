-- Binary integer literals: a deliberate golua extension.
--
-- Reference Lua defines only decimal and hexadecimal numerals, so this file
-- fails to COMPILE there ("malformed number near '0b101'"); golua runs it.

print(0b101)
--> golua:     5
--> lua5.5.0:  malformed number near '0b101' (compile error)

-- Detectable through load as a dialect probe:
print(load("return 0b1111") ~= nil)
--> golua:     true
--> lua5.5.0:  false
