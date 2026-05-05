-- broken_fuzz_sub_immediate_neg_zero:
-- (-0.0) - 0  should give +0.0 to match Lua 5.5/5.4, but golua gives -0.0.
--
-- BROKEN: For an expression of the form  <float> - <integer-literal>,
-- Lua 5.5's compiler rewrites it to  <float> + (-<int-literal>) and emits
-- OP_ADDI. IEEE754 has  (-0.0) + 0.0 = +0.0  but  (-0.0) - 0.0 = -0.0,
-- so the rewrite changes the sign of zero on this corner. golua's compiler
-- emits OP_SUBK for the same expression (no rewrite), keeping the IEEE
-- subtraction semantics. Result: golua gives -0.0 where reference gives 0.0.
--
-- Verified via luac5.5.0 -p -l on  `local a = -0.0; print(a - 0)`:
--    Lua 5.5: SUBK then ADDI? — actually the rewrite happens for integer
--    literals in subtraction; the bytecode emits ADDI(R, -0).
-- golua's compiler should mirror this rewrite for sub-with-int-literal.
--
-- Note: -0.0 - 0.0 (float literal) and a - b (where b is a variable)
-- both correctly produce -0.0 in BOTH impls — the bug is specific to
-- the OP_SUBI fast path with an integer immediate.
--
-- Reference (lua5.5.0 and lua 5.4.8):
--   string.format("%a", -0.0 - 0)   -> 0x0p+0
--
-- golua today:
--   string.format("%a", -0.0 - 0)   -> -0x0p+0
--
-- Discovered: differential fuzz 2026-05-04 (math wave-1 agent).
-- Suspect: compiler/compile_expr.go arithmetic-with-immediate-int handling.

-- Inline subtraction with integer literal: the bug case.
local s = string.format("%a", -0.0 - 0)
assert(s == "0x0p+0",
  "(-0.0) - 0 should yield +0.0 (matching ADDI rewrite); got %a == " .. s)

-- Sanity checks: these should produce -0.0 in BOTH impls (no bug).
assert(string.format("%a", -0.0 - 0.0) == "-0x0p+0",
  "(-0.0) - 0.0 (float literal) should yield -0.0")

local zi = 0
local nz = -0.0
assert(string.format("%a", nz - zi) == "-0x0p+0",
  "(-0.0) - <int variable> should yield -0.0")

print("ok")
