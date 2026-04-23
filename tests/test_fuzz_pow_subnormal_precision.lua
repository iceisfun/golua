-- Precision of `^` for subnormal (denormal) bases.
--
-- Background:
--   Go's math.Pow computes x^y as Exp(yf*Log(x)) * x**yi, which loses
--   precision catastrophically when x is subnormal because Log(x) is
--   inaccurate deep in the denormal range. For example:
--     (5e-324) ^ -0.69314718055994529
--       raw Go math.Pow  ->  6.45295550853252e+228
--       lua5.5.0/libm    ->  1.2554274566273629e+224
--     i.e. ~5.1e+4x off for this input.
--
-- Fix (see vm/vm_pow.go PowWithSubnormalFix):
--   For a positive subnormal x, decompose exactly as x = m * 2^-1074
--   where m is the raw mantissa (an integer in [1, 2^52-1], a normal
--   float). Then x^y = m^y * 2^(-1074*y), and both factors are computed
--   in the normal float64 range so libm-grade precision is preserved.
--   Wired into OP_POW/OP_POWK (vm/vm_arith.go), constant folding
--   (compiler/compile_expr.go), math.pow (stdlib/math.go), and the
--   string metatable __pow (stdlib/string.go).

local a = 5e-324
local b = -0.69314718055994529
local got = a ^ b
local want = 1.2554274566273629e+224
-- Require result within 1% of libm's value.
assert(math.abs(got - want) / want < 0.01,
  string.format("pow subnormal precision: got=%.17e want=%.17e", got, want))

-- Additional sanity cases (compared against lua5.5.0 / libm):
local function close(x, y, rel)
  if x == y then return true end
  if y == 0 then return math.abs(x) < 1e-300 end
  return math.abs(x - y) / math.abs(y) < (rel or 0.01)
end

assert(close((5e-324) ^ -0.5, 4.49891379454319638e+161))
assert((5e-324) ^ 2 == 0.0)
-- negative subnormal with integer odd y stays signed zero
local nz = (-5e-324) ^ 3
assert(nz == 0.0)
