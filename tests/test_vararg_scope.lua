-- Bug: ... is allowed in non-vararg functions.
-- Lua 5.4 rejects ... at compile time if used outside a vararg function.

-- ... in a non-vararg function should fail at compile time
local ok, err = pcall(load, "function f(a, b) return ... end")
assert(not ok or (ok and err == nil),
  "load should fail for ... in non-vararg function")
-- If load returns a function, the compilation wrongly succeeded
if type(ok) == "boolean" and ok and err ~= nil then
  error("... was accepted in non-vararg function but should be rejected")
end

-- Actually, load returns (nil, errmsg) on compile error, or (func, nil) on success
local f, errmsg = load("function f(a, b) return ... end")
assert(f == nil, "... in non-vararg function should be a compile error, but it compiled")
assert(type(errmsg) == "string" and errmsg:find("%.%.%."),
  "error should mention '...', got: " .. tostring(errmsg))

print("PASS")
