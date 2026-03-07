-- Test that table constructors with record fields do not clobber the table
-- register when freeReg <= reg (e.g. after local y = x; x = {key=val}).

-- Minimal repro: reassign a local to a record-style table constructor
-- when another local holds the old value (so freeReg can lag behind reg).
local x = 1
local y = x
x = {key = "val"}
assert(type(x) == "table", "expected table, got " .. type(x))
assert(x.key == "val", "expected 'val', got " .. tostring(x.key))

-- Multiple record fields
local a = 0
local b = a
a = {one = 1, two = 2, three = 3}
assert(a.one == 1)
assert(a.two == 2)
assert(a.three == 3)

-- Nested table constructor
local p = nil
local q = p
p = {inner = {deep = true}}
assert(type(p.inner) == "table")
assert(p.inner.deep == true)

-- Mixed array + record
local m = 0
local n = m
m = {10, 20, key = "val"}
assert(m[1] == 10)
assert(m[2] == 20)
assert(m.key == "val")

-- In a loop
local prev = nil
for i = 1, 5 do
    local old = prev
    prev = {val = i}
end
assert(prev.val == 5)

