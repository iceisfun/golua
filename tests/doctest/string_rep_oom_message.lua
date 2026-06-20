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

-- A single-byte string repeated math.maxinteger times yields a total length of
-- exactly math.maxinteger (== MAX_SIZE on 64-bit). Reference Lua's buffer
-- allocator rejects a >= MAX_SIZE request with "resulting string too large"
-- before attempting any allocation, even though the size is representable.
-- The "len + lsep > MAX_SIZE/n" overflow guard does not catch this case
-- (1 > maxinteger/maxinteger == 1 is false), so it needs its own check.
print(pcall(string.rep, "x", math.maxinteger))
--> =false	resulting string too large

-- One repetition fewer is representable but unallocatable: "not enough memory".
print(pcall(string.rep, "x", math.maxinteger - 1))
--> =false	not enough memory

-- A two-byte string at maxinteger overflows the size computation: "too large".
print(pcall(string.rep, "xy", math.maxinteger))
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
