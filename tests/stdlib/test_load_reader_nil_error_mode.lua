-- load: a nil error from the reader and the mode check on the first piece.

-- a reader raising a nil error reports "<no error object>"
local ok, f, e = pcall(load, error)
assert(ok and f == nil and e == "<no error object>", tostring(e))
local f2, e2 = coroutine.wrap(load)(error)
assert(f2 == nil and e2 == "<no error object>", tostring(e2))

-- mode is checked on the first piece, the reader is not drained
local n = 0
local f3, e3 = load(function() n = n + 1; if n < 5 then return "return 1 " end end, "nm", "b")
assert(f3 == nil and e3 == "attempt to load a text chunk (mode is 'b')", e3)
assert(n == 1, n)
assert(select(2, load(function() return "return 1 " end, "nm", "b"))
  == "attempt to load a text chunk (mode is 'b')")

print("OK")
