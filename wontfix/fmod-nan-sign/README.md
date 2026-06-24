# fmod-nan-sign

## What

`math.fmod` and the `%` operator can produce a NaN whose **sign bit** differs
from reference Lua's, which shows up as `-nan` vs `nan` when printed:

```lua
math.fmod(2, -(0/0))   -- golua: -nan   reference: nan
```

The result is NaN in both cases — only the (semantically meaningless) sign of
the NaN differs, and only for a finite dividend with an explicitly *negative*
NaN divisor. For every ordinary NaN source (`0/0`, `x % 0`, `fmod(x, 0/0)`)
golua matches the reference.

## Why this won't change

The sign of a NaN carries no value — IEEE 754 leaves it unspecified for most
operations, and C's `fmod` propagates whatever the platform libm chooses. golua
forces a NaN sign in a few mod/fmod paths specifically to match glibc's *usual*
behavior, but glibc's choice for this particular `finite mod -NaN` input differs
from the forced sign, and there is no portable rule that matches every libm.

Chasing exact NaN-sign parity would mean reverse-engineering one platform's
libm sign conventions for every NaN-producing input — a moving, non-portable
target with zero observable effect on programs (`tostring` aside, a `-nan` and a
`nan` behave identically: not equal to anything, not ordered, etc.).

## Where this lives in the source

- [`vm/vm_arith.go`](../../vm/vm_arith.go) — `luaNumMod` (the `%` operator).
- [`stdlib/math.go`](../../stdlib/math.go) — `math.fmod` and related functions
  (`math.Copysign(math.NaN(), -1)` sites).

## Related

Surfaced by the math/coercion sweeps in `golua-conformance`; classified there as
a C-libm-dependent NaN-sign platform difference.
