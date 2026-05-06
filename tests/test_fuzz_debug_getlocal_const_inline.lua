-- broken_fuzz_debug_getlocal_const_inline:
-- <const> locals initialized with foldable scalars are exposed by
-- debug.getlocal in golua. Reference Lua 5.5 inlines them at compile
-- time so they consume no register and are NOT visible via getlocal.
--
-- BROKEN: compiler — when a `local x <const> = <foldable_scalar>` is
-- declared, the compiler should inline x at use-sites and not allocate
-- a register or a debug-info entry. golua currently reserves a register
-- and exposes the local, shifting all subsequent local indices.
--
-- Affects: number, string, boolean, nil, and constant-foldable
-- expressions. Non-foldable initializers (function, table) are
-- correctly NOT inlined in either impl.
--
-- Reference (lua5.5.0):
--   local function f()
--     local x <const> = 42
--     local y = 100
--     for i = 1, 4 do ...; n, v = debug.getlocal(1, i); ... end
--   end
--   First emit: y, 100 (x is inlined)
--
-- golua today:
--   First emit: x, 42 (x still has a register and a debug entry)
--   This shifts indices for every consumer iterating debug.getlocal
--   by 1.
--
-- Discovered: differential fuzz 2026-05-04 (debug wave-3 agent).

local function f()
  local x <const> = 42
  local y = 100
  -- The first variable visible to debug.getlocal should be y (x is inlined).
  local n, v = debug.getlocal(1, 1)
  return n, v
end

local n, v = f()
assert(n == "y", "first local should be 'y' (x is inlined); got " .. tostring(n))
assert(v == 100, "first local value should be 100; got " .. tostring(v))

print("ok")
