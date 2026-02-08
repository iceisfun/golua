-- BROKEN: Frontier pattern (%f[]) not yet implemented
-- Tracking: Phase 2 (pattern engine completion)
--
-- The frontier pattern %f[set] matches at a position where the character
-- before does not match [set] but the character after does. This is used
-- for word boundary matching and similar tasks.

local r = string.find("THE END", "%f[%a]%u+")
assert(r == 5, "frontier pattern should match 'END' at position 5")
