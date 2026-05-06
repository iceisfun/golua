-- broken_fuzz_named_vararg_leading_nil_rawlen:
-- Named-vararg parameter `function f(...args)` (Lua 5.5 feature) packs
-- the varargs into a table. When the FIRST vararg is nil, golua's
-- packing skips the array growth and the resulting table has rawlen == 0
-- instead of the expected count. Reference Lua 5.5 packs identically to
-- table.pack: array part holds all values 1..n regardless of nil.
--
-- BROKEN: vm/vm_exec.go around line 1830 (createVarArgTable) uses
--   t.SetInt(i+1, val)
-- which is the metamethod-aware setter. SetInt does NOT extend the array
-- part when storing nil at the next array slot — and a nil at index 1
-- means index 1 stays unset, so subsequent SetInt calls for indices
-- 2..n land in the hash part (since the array part is still empty,
-- index 2 isn't len+1 anymore — it's len(0)+2).
--
-- Reference (lua5.5.0):
--   function g(...args) return rawlen(args), args.n end
--   print(g(nil, 2, 3, 4, 5))
--   -> 5    5     (rawlen reflects the packed array)
--
-- golua today:
--   -> 0    5     (n is correct but rawlen is wrong)
--
-- Note: 5.4 doesn't have named vararg; this is a 5.5-specific feature.
-- The fix is to use the raw array setter (e.g., RawSetArray or equivalent
-- internal method that grows the array even for nil at position len+1)
-- when packing named varargs, similar to how table.pack must handle nil
-- packing.
--
-- Discovered: differential fuzz 2026-05-04 (vararg+select wave-3 agent).

local function g(...args)
  return rawlen(args), args.n, args[1], args[5]
end

local rl, n, first, last = g(nil, 2, 3, 4, 5)
assert(n == 5, "args.n must be 5")
assert(first == nil, "args[1] should be nil")
assert(last == 5, "args[5] should be 5")
assert(rl == 5, "rawlen(args) should be 5 to match table.pack-like packing; got " .. rl)

print("ok")
