-- When a gmatch iterator is exhausted, it should return 0 values (not 1 nil).

-- Basic test: select('#', g()) should be 0 after exhaustion
local g = ("abc"):gmatch("(%a)")
assert(g() == "a")
assert(g() == "b")
assert(g() == "c")
assert(select("#", g()) == 0, "exhausted gmatch should return 0 values")

-- Without captures: same behavior
local g2 = ("xy"):gmatch("%a")
assert(g2() == "x")
assert(g2() == "y")
assert(select("#", g2()) == 0, "exhausted gmatch (no captures) should return 0 values")

-- Multiple calls after exhaustion should all return 0 values
assert(select("#", g2()) == 0, "repeated calls after exhaustion should return 0 values")
assert(select("#", g2()) == 0, "third call after exhaustion should return 0 values")

-- Empty string: immediately exhausted
local g3 = (""):gmatch("%a")
assert(select("#", g3()) == 0, "gmatch on empty string should return 0 values immediately")

-- for-in loop should terminate correctly (this uses the 0-return to stop)
local count = 0
for w in ("hello"):gmatch("%a") do
    count = count + 1
end
assert(count == 5, "for-in with gmatch should iterate correctly")

print("OK")
