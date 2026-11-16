-- Test: utf8.codes iterator driven by hand
-- What: like reference iter_aux, the control value is a byte offset from which
-- continuation bytes are skipped, and a finished iteration returns no values

local f, s = utf8.codes("ab")
assert(select("#", f(s, 2)) == 0)
assert(select("#", f(s, 5)) == 0)
assert(select("#", f(s, -1)) == 0)
assert(select("#", utf8.codes("")("", 0)) == 0)

local g, t = utf8.codes("\u{20AC}x")   -- 3-byte character, then 'x'
local p, c = g(t, 1)
assert(p == 4 and c == 120)
p, c = g(t, 2)                           -- control inside the character
assert(p == 4 and c == 120)
p, c = g(t, 3)
assert(p == 4 and c == 120)

local h, u = utf8.codes("a\u{20AC}")
assert(select("#", h(u, 3)) == 0)       -- inside the last character
assert(select("#", h(u, 4)) == 0)

-- a plain loop is unchanged
local out = {}
for pos, cp in utf8.codes("a\u{20AC}b") do out[#out + 1] = pos .. ":" .. cp end
assert(table.concat(out, ",") == "1:97,2:8364,5:98")

-- invalid sequences still raise
local ok = pcall(function() for _ in utf8.codes("a\xE2\x82") do end end)
assert(not ok)
