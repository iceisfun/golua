# untrusted-binary-chunks

## What

`load(chunk, name, "b")` — or the default `"bt"` mode when `chunk` begins with
the Lua binary signature — deserializes and then executes **raw bytecode**.
Executing a *maliciously crafted* binary chunk can hang or crash the
interpreter. This is true of golua and of reference Lua alike.

## Why this won't change

The Lua reference manual (§6.1, `load`) states plainly:

> Lua does not check the consistency of binary chunks. Maliciously crafted
> binary chunks can crash the interpreter.

Reference Lua ships **no** bytecode verifier; defending the VM against every
adversarial proto (out-of-range registers/constants, bad jumps, enormous loop
bounds, malformed line tables) would mean building one golua-side that the
reference does not have, at a real cost to every legitimate `load`. The
load-time *and* the run-time behavior here track the reference's documented
stance, so matching it is the correct baseline.

## What golua *does* harden

Loading must never be an **uncatchable** host crash, even though execution may
be unsafe:

- A corrupt element count in the binary header can no longer drive an unbounded
  `make([]T, count)` into a Go `runtime.throw` fatal OOM (which `recover()`
  cannot catch). `compiler/undump.go`'s `readCount` bounds every
  allocation-driving count by the bytes remaining in the chunk, so a malformed
  chunk surfaces as a catchable `"bad binary format (truncated chunk)"` error.
  See the regression test `TestUndumpHugeCountIsCatchable`.

So: **loading** untrusted binary is safe (catchable errors, bounded memory);
**executing** untrusted binary is not, by design and in agreement with the
reference.

## Mitigation for sandboxes

Do not expose binary chunk loading to untrusted code. An embedder that calls
untrusted source should restrict `load` to **text mode** (`"t"`), exactly as
recommended for reference Lua. Sandboxed *source* Lua (golua's actual
robustness guarantee) cannot reach this path.

## Where this lives in the source

- Allocation bound: [`compiler/undump.go`](../../compiler/undump.go) — `readCount`.
- Regression pin: `TestUndumpHugeCountIsCatchable` in
  [`compiler/undump_security_test.go`](../../compiler/undump_security_test.go).
