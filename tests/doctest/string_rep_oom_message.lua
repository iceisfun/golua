-- string.rep distinguishes two failure modes like reference Lua:
--   * "not enough memory" when the resulting size is representable but cannot
--     be allocated (this is what users hit for merely-huge counts).
--   * "resulting string too large" only when the size computation overflows.
-- Previously golua reported "resulting string too large" for every oversized
-- request, diverging from Lua 5.5 which reports "not enough memory" for these
-- representable-but-unallocatable sizes.

print(pcall(string.rep, "ab", 1000000000000))
--> =false	not enough memory

print(pcall(string.rep, "abcdefghij", 1000000000))
--> =false	not enough memory

print(pcall(string.rep, "ab", 1000000000000, "sep"))
--> =false	not enough memory

-- a genuine size-computation overflow still reports "too large"
print(pcall(string.rep, "x", math.maxinteger, "y"))
--> =false	resulting string too large

-- ordinary uses are unaffected
print(string.rep("ab", 3, "-"))
--> =ab-ab-ab

print(string.rep("x", 0))
--> =

print((pcall(string.rep, "ab", -5)))
--> =true

-- non-positive count yields the empty string
print(string.rep("ab", -5) == "")
--> =true
