-- Test that OP_VARARG with many varargs grows the stack properly.
-- Previously, expanding 1000+ varargs could cause a Go panic when
-- the stack had grown to 1024 slots but needed more.
local a = {}
for i = 1, 1000 do a[i] = i end

local function foo(first, ...)
  local t = table.pack(...)
  assert(t.n == 999, "expected 999 varargs, got " .. t.n)
  assert(first == 1, "first arg")
  assert(t[1] == 2, "first vararg")
  assert(t[999] == 1000, "last vararg")
end

foo(table.unpack(a))
print("PASS") --> PASS
