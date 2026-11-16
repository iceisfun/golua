-- Test: string.rep with an empty result returns at once
-- What: s and sep both empty means the result is empty whatever n is, so the
-- huge count must not be iterated (reference str_rep returns "" immediately)

assert(string.rep("", math.maxinteger) == "")
assert(string.rep("", 1 << 50, "") == "")
assert(string.rep("", 100000000, "") == "")
assert(#string.rep("", math.maxinteger, "") == 0)

-- non-empty results are unchanged
assert(string.rep("ab", 3) == "ababab")
assert(string.rep("", 3, ",") == ",,")
assert(string.rep("a", 0, ",") == "")
