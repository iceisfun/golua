-- broken_fuzz_upvaluejoin_c_function: upvaluejoin rejects C functions
--
-- BROKEN: `debug.upvaluejoin(f1, n1, f2, n2)` in Lua 5.5 accepts any callable
-- for f1/f2; the relevant validation is that both upvalue indices are valid.
-- golua rejects C functions outright with an over-strict "Lua function
-- expected" check, and also reports the wrong argument number in the error
-- message (#1/#3 vs reference #2/#4).
--
-- Reference (lua5.5.0):
--   debug.upvaluejoin(print, 1, f, 1) → error at arg #4 "(invalid upvalue index)"
--                                      (print has no upvalue #1)
-- golua today:
--   error at arg #3 "(Lua function expected, got function)"
--
-- Fix: stdlib/debug.go `luaDebugUpvalueJoin` — remove the type check, keep
-- only upvalue-index validation; fix arg-number in error to match reference
-- (f-arg is #1 or #3, upvalue-index is #2 or #4).
--
-- Discovered: differential fuzz 2026-04-23 (debuglib_4).

local function f() local x = 1; return function() return x end end
local g = f()

-- C function on the source side — must report arg #4, "invalid upvalue index"
local ok1, err1 = pcall(debug.upvaluejoin, g, 1, print, 1)
assert(not ok1)
assert(tostring(err1):find("invalid upvalue index"),
       "expected 'invalid upvalue index', got: " .. tostring(err1))
assert(tostring(err1):find("#4"),
       "expected arg #4, got: " .. tostring(err1))

-- C function on the destination side — arg #2, "invalid upvalue index"
local ok2, err2 = pcall(debug.upvaluejoin, print, 1, g, 1)
assert(not ok2)
assert(tostring(err2):find("invalid upvalue index"),
       "expected 'invalid upvalue index', got: " .. tostring(err2))
assert(tostring(err2):find("#2"),
       "expected arg #2, got: " .. tostring(err2))
