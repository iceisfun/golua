-- BUG: Native function == comparison panics
-- In Lua 5.4, native (C) functions can be compared with == just like
-- closures. They are equal if they are the same function object.
-- GoLua panics with "comparing uncomparable type vm.NativeFunc".

-- Same native function should be equal
assert(print == print, "print == print should be true")
-- Different native functions should not be equal
assert(print ~= tostring, "print ~= tostring should be true")
-- Native function compared with non-function should be false
assert(print ~= 42, "print ~= 42 should be true")
assert(print ~= "hello", "print ~= 'hello' should be true")
assert(print ~= nil, "print ~= nil should be true")
-- Native functions stored in tables should be comparable
local t = {f = print}
assert(t.f == print, "t.f == print should be true")
