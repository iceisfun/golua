# libm-last-ulp

## What

Transcendental and irrational floating-point functions can differ from the
reference interpreter in the **last unit in the last place (ULP)** of the
result, for some inputs:

```lua
string.format("%.17g", 2.5 ^ 131)
-- golua:    1.3494013367335074e+52
-- lua5.5.0: 1.3494013367335069e+52
```

Affected operations include `^` (pow), `math.sin/cos/tan`, `math.asin/acos/atan`,
`math.exp`, `math.log`, and `math.sqrt` for non-exact arguments. Exact results
(`2^53`, `math.sqrt(4)`, integer powers) are identical on both.

### Large-argument trig is *more* than last-ULP

For trig functions of **large** arguments the difference grows well past one ULP
— hundreds to thousands of ULP — because the result depends on *argument
reduction* (computing `x mod 2π` for a huge `x`), which Go's `math` and the
platform libm implement differently:

```lua
string.format("%a", math.tan(0x1.0b7f1eae38000p+39))  -- ~86 ULP apart
string.format("%a", math.cos(0x1.6013f070d1452p+927)) -- ~1270 ULP apart
```

This is the *same* Go-vs-libm root cause, not a distinct bug. It is also
mathematically unavoidable: once `|x|` exceeds `2^52`, consecutive doubles are
spaced more than `2π` apart, so the stored value cannot even identify which
period it lands in — there is no meaningful "correct" `tan`/`cos` to converge
on. `tan` is the worst because its poles amplify any reduction error. The
`mathfuzz` finder therefore feeds random doubles only to the *exact* functions
(`sqrt`/`floor`/`ceil`/`abs`/`modf`/…), never to transcendentals, so it does not
manufacture this expected noise.

## Why this won't change

golua computes these with Go's [`math`](https://pkg.go.dev/math) package;
reference Lua calls into the platform C library (libm, typically glibc). Both are
high-quality and conform to IEEE 754's accuracy expectations (correctly rounded
to ≤ ~1 ULP), but they are *different implementations* and disagree on the final
bit for some inputs. There is no "more correct" answer to converge on — the true
result is irrational and both round it legitimately.

Matching the reference bit-for-bit would require golua to ship and call its own
copy of a specific libm version, which contradicts being a pure-Go
implementation and would still only match *one* platform's libm. We accept the
last-ULP difference.

## Where this lives in the source

- [`stdlib/math.go`](../../stdlib/math.go) — every `math.*` function delegates to
  Go's `math` package.
- The `^` operator: the VM arithmetic path, [`vm/vm_arith.go`](../../vm/vm_arith.go).

## Related

Differential finders `formatfuzz`, `coercionfuzz`, and the math sweeps in
`golua-conformance` treat last-ULP transcendental differences as a platform
won't-fix and corroborate that a flagged math difference is *only* last-ULP
before reporting it.
