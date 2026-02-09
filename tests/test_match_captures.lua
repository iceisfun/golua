-- string.match should return captures when pattern has capture groups
assert(string.match("Let me *stress* that I *am*", "%*(.-)%*") == "stress",
    "match should return capture, not full match")

-- Multiple captures
local d, w = string.match("A *bold* and an _underline_", "([*~_])(.-)%1")
assert(d == "*" and w == "bold", "match should return back-reference captures")

-- Init argument
assert(string.match("Let me *stress* that I *am*", "%*(.-)%*", 17) == "am",
    "match should support init argument")
