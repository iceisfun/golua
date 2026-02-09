-- test_string_match2: string.match basic functionality (no captures)

-- Basic match without captures returns the whole match
assert(string.match("Let me *stress* that I *am*", "%*.-%*") == "*stress*",
    "match non-greedy")

-- Negative start position
assert(string.match("abcd", "b", -100) == "b", "match negative start")

-- No match returns nil
assert(string.match("abc", "xyz") == nil, "match no match")

-- Error on missing args
assert(not pcall(string.match, "x"), "match needs 2 args")
