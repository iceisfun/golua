-- Test: f:read with several formats stops at the first failure and returns
-- exactly one nil for it; read('n') skips \v and \f like C isspace.
local name = os.tmpname()
local f = io.open(name, "w") f:write("a") f:close()

f = io.open(name)
assert(select('#', f:read('l', 'l', 'l', 'l')) == 2)
f:close()

f = io.open(name)
local a, b, c = f:read('l', 'l', 'l')
assert(a == "a" and b == nil and c == nil)
assert(select('#', f:read('l', 'L', 'n')) == 1)
f:close()

f = io.open(name, "w") f:close()
f = io.open(name)
assert(select('#', f:read('n', 'l')) == 1)
f:close()

for _, c in ipairs{"\v", "\f", " ", "\t"} do
  f = io.open(name, "wb") f:write(c .. "3") f:close()
  f = io.open(name)
  assert(f:read('n') == 3, "leading whitespace " .. string.format("%q", c))
  f:close()
end

os.remove(name)
print("PASS")
