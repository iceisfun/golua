-- Bug #1: Right shift (>>) does arithmetic shift (sign-extending) instead of logical (zero-filling)
-- Lua 5.4 specifies logical shifts: vacated bits are filled with zeros.

assert(-1 >> 1 == 0x7FFFFFFFFFFFFFFF)    -- should zero-fill from the left
assert(-1 >> 63 == 1)                     -- only the former sign bit remains
assert(math.mininteger >> 1 == 0x4000000000000000) -- 0x8000... >> 1 = 0x4000...

-- Identity and negative-count shifts should still work
assert(-1 >> 0 == -1)                     -- shift by 0 is identity
assert(1 << -1 == 0)                      -- negative left shift = right shift
assert(1 >> -1 == 2)                      -- negative right shift = left shift
