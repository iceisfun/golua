-- test_utf8_lax.lua
-- utf8.len/codepoint should accept the lax flag and behave identically on valid data.

local sample = "héllo"

local len_strict = utf8.len(sample)
local len_lax = utf8.len(sample, 1, -1, true)
assert(len_strict == len_lax, string.format("expected lax len to equal strict len (%d vs %d)", len_strict, len_lax))

local c1, c2 = utf8.codepoint(sample, 1, 2)
local l1, l2 = utf8.codepoint(sample, 1, 2, true)
assert(c1 == l1 and c2 == l2, "utf8.codepoint should accept lax flag on valid data")
