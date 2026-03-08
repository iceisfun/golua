-- Test io.read("n") with hex floats, partial matches, and whitespace-only input

-- Test 1: hex float 0x1.8p1 should parse as 3.0
local fname = os.tmpname()
local f = io.open(fname, "w")
f:write("0x1.8p1")
f:close()

f = io.open(fname, "r")
local v = f:read("n")
f:close()
os.remove(fname)

assert(v == 3.0, "expected 3.0 for 0x1.8p1, got " .. tostring(v))
print("PASS: hex float 0x1.8p1 => " .. tostring(v))

-- Test 2: "42abc" should parse as 42, stopping at 'a'
f = io.open(fname, "w")
f:write("42abc")
f:close()

f = io.open(fname, "r")
v = f:read("n")
-- After reading 42, position should be at 2 (just past "42")
local rest = f:read("a")
f:close()
os.remove(fname)

assert(v == 42, "expected 42 for '42abc', got " .. tostring(v))
assert(rest == "abc", "expected remaining 'abc', got " .. tostring(rest))
print("PASS: partial '42abc' => " .. tostring(v) .. " rest='" .. rest .. "'")

-- Test 3: whitespace-only "   " should return nil and leave position after spaces
f = io.open(fname, "w")
f:write("   ")
f:close()

f = io.open(fname, "r")
v = f:read("n")
local pos = f:seek("cur")
f:close()
os.remove(fname)

assert(v == nil, "expected nil for whitespace-only, got " .. tostring(v))
assert(pos == 3, "expected position 3 after reading spaces, got " .. tostring(pos))
print("PASS: whitespace-only => nil, pos=" .. tostring(pos))

-- Test 4: hex float 0xABp2
f = io.open(fname, "w")
f:write("0xABp2")
f:close()

f = io.open(fname, "r")
v = f:read("n")
f:close()
os.remove(fname)

assert(v == 0xABp2, "expected " .. tostring(0xABp2) .. " for 0xABp2, got " .. tostring(v))
print("PASS: hex float 0xABp2 => " .. tostring(v))

-- Test 5: plain integer
f = io.open(fname, "w")
f:write("  123  ")
f:close()

f = io.open(fname, "r")
v = f:read("n")
f:close()
os.remove(fname)

assert(v == 123, "expected 123, got " .. tostring(v))
print("PASS: plain integer => " .. tostring(v))

-- Test 6: negative number
f = io.open(fname, "w")
f:write("-7.5")
f:close()

f = io.open(fname, "r")
v = f:read("n")
f:close()
os.remove(fname)

assert(v == -7.5, "expected -7.5, got " .. tostring(v))
print("PASS: negative float => " .. tostring(v))

print("All read('n') tests passed!")
