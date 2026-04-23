-- Lua 5.5: debug.upvaluejoin accepts any callable (including C functions),
-- and only the upvalue indices are validated. The arg number in the error
-- message refers to the upvalue-index argument (#2 or #4), not the function.
--
-- golua previously rejected C functions with "Lua function expected" and
-- reported the function's arg number instead of the upvalue-index arg.

local g = function() local x = 1; return function() return x end end
local inner = g()  -- closure with one upvalue (x)

-- C function (print) as f2 with invalid upvalue index: should fail at arg #4.
local ok, err = pcall(debug.upvaluejoin, inner, 1, print, 1)
assert(not ok, "should fail")
assert(err:find("bad argument #4 to 'debug.upvaluejoin'") ~= nil,
  "expected arg #4 error, got: " .. tostring(err))
assert(err:find("invalid upvalue index") ~= nil,
  "expected 'invalid upvalue index', got: " .. tostring(err))

-- C function (print) as f1 with invalid upvalue index: should fail at arg #2.
local ok2, err2 = pcall(debug.upvaluejoin, print, 1, inner, 1)
assert(not ok2, "should fail")
assert(err2:find("bad argument #2 to 'debug.upvaluejoin'") ~= nil,
  "expected arg #2 error, got: " .. tostring(err2))
assert(err2:find("invalid upvalue index") ~= nil,
  "expected 'invalid upvalue index', got: " .. tostring(err2))

print("PASSED")
