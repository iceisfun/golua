-- BROKEN: Precision divergence in `^` for subnormal (denormal) bases.
--
-- Example:
--   (5e-324) ^ -0.69314718055994529
--     golua    ->  6.45295550853252e+228
--     lua5.5.0 ->  1.2554274566273629e+224   (matches system libm pow)
--   Error ~ 5.1e+4x (many orders of magnitude) for subnormal inputs.
--   For normal-magnitude bases the divergence is typically 1-2 ULPs.
--
-- Root cause: this is a Go-stdlib-vs-libm issue, not a bug in golua.
-- golua's `^` operator delegates to Go's `math.Pow` (see vm/vm_arith.go
-- cases OP_POW/OP_POWK, compiler/compile_expr.go constant folding, and
-- stdlib/math.go for math.pow). Go's math.Pow (src/math/pow.go) computes:
--       ans = Exp(yf * Log(x)) * x**yi     (yi integer part, yf fractional)
-- For x = 5e-324 (smallest positive subnormal), Log(x) already carries
-- large relative error (log loses precision deep in the denormal range),
-- and multiplying by y then exponentiating amplifies that error
-- exponentially. Net result: ~4 decimal orders of magnitude off.
--
-- libm's pow (glibc/FDLIBM) uses an extended-precision (double-double or
-- split-polynomial) decomposition of log2(x) and exp2, so it stays
-- faithfully rounded even for subnormal x. Lua 5.5 links directly to
-- libm's pow, so `lua5.5.0` matches libm exactly.
--
-- Options evaluated (2026-04-23):
--   (a) Accept divergence, keep skipped. CHOSEN. This is a known intrinsic
--       limitation of Go's math.Pow for subnormal bases; fixing it requires
--       a bit-identical pow implementation.
--   (b) Workaround via `exp(b * log(a))`. REJECTED: that *is* what Go
--       already does internally, and naive reimplementation gives a
--       different wrong answer (verified: ~2.87e+213, also far from libm).
--       A correct workaround requires extended precision.
--   (c) CGo bridge to libm's pow. REJECTED: the project avoids CGo;
--       adding it for one operator is disproportionate and breaks
--       cross-compilation.
--   (d) Vendor a bit-identical pow (e.g. musl or FDLIBM). REJECTED for
--       now: non-trivial port (~300 LoC of numerically sensitive code),
--       risks regressing the common-case pow tests that currently pass,
--       and would need extensive differential testing against libm across
--       the full float64 domain to be trustworthy.
--
-- If/when a correctly-rounded pow lands in Go's stdlib (tracked upstream
-- in golang/go issues around math precision) or a well-tested Go port is
-- available, revisit option (d), rename this file to
-- test_fuzz_pow_subnormal_precision.lua, and enable.

local a = 5e-324
local b = -0.69314718055994529
local got = a ^ b
local want = 1.2554274566273629e+224
-- Require result within 1% of libm's value.
assert(math.abs(got - want) / want < 0.01,
  string.format("pow subnormal precision: got=%.17e want=%.17e", got, want))
