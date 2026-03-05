-- gmatch with ^ anchor should not strip the ^ from the pattern.
-- In Lua 5.4, ^ is not treated as an anchor in gmatch; it's a literal character
-- that gets passed to the pattern matcher and fails to match any input character.

-- Basic: ^%a+ should produce 0 matches (^ is literal, not anchor)
local c = 0
for w in ("hello world"):gmatch("^%a+") do
    c = c + 1
end
assert(c == 0, "expected 0 matches for ^%a+, got " .. c)

-- ^hello should also produce 0 matches
c = 0
for w in ("hello world"):gmatch("^hello") do
    c = c + 1
end
assert(c == 0, "expected 0 matches for ^hello, got " .. c)

-- ^. should produce 0 matches (^ is literal)
c = 0
for w in ("abc"):gmatch("^.") do
    c = c + 1
end
assert(c == 0, "expected 0 matches for ^., got " .. c)

-- Pattern without ^ should still work normally
c = 0
for w in ("hello world"):gmatch("%a+") do
    c = c + 1
end
assert(c == 2, "expected 2 matches for %a+, got " .. c)

-- Escaped ^ (%^) should match literal ^
c = 0
local results = {}
for w in ("a^b^c"):gmatch("[^%^]+") do
    c = c + 1
    results[c] = w
end
assert(c == 3, "expected 3 matches for [^%%^]+, got " .. c)
assert(results[1] == "a")
assert(results[2] == "b")
assert(results[3] == "c")

print("OK")
