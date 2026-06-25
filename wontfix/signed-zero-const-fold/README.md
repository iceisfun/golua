# signed-zero-const-fold

## What

When an arithmetic expression over numeric **constants** evaluates to negative
zero, golua's compile-time constant folder preserves the IEEE 754 sign:

```lua
string.format("%.17g", ((-0.0)) - (0))   -- golua: -0   reference: 0
```

Reference Lua's constant folder produces `+0.0` here. The same expression
computed from **runtime** values (not folded) agrees on both implementations.

## Why this won't change

golua is the *more* correct of the two here. Under IEEE 754:

- `(-0.0) - (+0.0)` is `-0.0` (subtracting +0 never flips a sign).

Reference Lua's folder normalizes this to `+0.0` as an artifact of how it
reuses its number-formatting path during folding; it is not a guarantee Lua
programs can rely on (the unfolded form already differs). Matching it would mean
deliberately discarding a correct sign bit in the compiler, which would also
diverge from what golua produces for the identical computation at runtime.

`-0.0` and `+0.0` are equal under `==`, hash to the same table key, and differ
only in `1/x` (→ `-inf` vs `+inf`) and in `%.17g`/`%a` formatting — so this has
no effect on ordinary arithmetic.

## Where this lives in the source

- Constant folding: [`compiler/compile_expr.go`](../../compiler/compile_expr.go)
  — `foldArith`, `foldUnaryArith`, `tryFoldConstScalar`.
- Regression test pinning golua's IEEE behavior:
  `TestConstantFoldNegativeZero` in
  [`compiler/compiler_test.go`](../../compiler/compiler_test.go).

## Related

Differential finder: `coercionfuzz` in `golua-conformance` (allowlisted there as
a platform won't-fix so its corpus reaches empty).
