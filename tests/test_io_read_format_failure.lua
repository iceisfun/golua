-- Test that read format failures stop subsequent reads (Lua 5.4 behavior)
-- When read("n") fails on non-numeric data, all following formats return nil

local fname = os.tmpname()
local f = io.open(fname, "w")
f:write("a line\nanother line\n1234\n3.45\none\ntwo\nthree\n")
f:close()

-- read("l", "n", "n", "l"): "n" fails on "another line", rest should be nil
f = io.open(fname, "r")
local l1, n1, n2, dummy = f:read("l", "n", "n", "l")
assert(l1 == "a line", "l1 should be 'a line', got: " .. tostring(l1))
assert(n1 == nil, "n1 should be nil (not a number), got: " .. tostring(n1))
assert(n2 == nil, "n2 should be nil (stopped after failure), got: " .. tostring(n2))
assert(dummy == nil, "dummy should be nil (stopped after failure), got: " .. tostring(dummy))
f:close()
print("format failure stops reads: OK")

-- Successful multi-format read
f = io.open(fname, "r")
l1, _, n1, n2 = f:read("l", "l", "n", "n")
assert(l1 == "a line", "l1=" .. tostring(l1))
assert(n1 == 1234, "n1=" .. tostring(n1))
assert(n2 == 3.45, "n2=" .. tostring(n2))
f:close()
print("multi-format success: OK")

os.remove(fname)
print("PASS")
