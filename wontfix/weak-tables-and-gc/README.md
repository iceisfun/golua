# weak-tables-and-gc

## What

Anything that depends on **when** an unreferenced value is collected differs
from reference Lua:

- Weak tables (`__mode = "k"`/`"v"`/`"kv"`) — entries are reclaimed on the Go
  GC's schedule, not synchronously with `collectgarbage()`.
- `__gc` finalizers — run when the Go runtime finalizes the object, in Go's
  order and timing, not Lua's.
- `collectgarbage("count")` deltas, step/pause/incremental knobs — golua maps
  `collectgarbage` onto Go GC *hints*; the exact numbers and stepping are not
  reproducible.

The value semantics still hold (a weak entry *will* eventually clear once
unreachable; finalizers *do* run) — only the timing and ordering differ.

## Why this won't change

golua deliberately delegates memory management to the Go runtime instead of
re-implementing Lua's incremental mark-and-sweep collector. That is a core
design choice: it makes golua safe to embed in Go programs, lets Lua values and
Go values share one heap, and avoids a second GC fighting the first.

Reference Lua's weak-table and finalizer guarantees are phrased around its own
collector's phases ("after a collection cycle…"). Those phases have no exact
equivalent in Go's concurrent collector. Forcing Lua-style synchronous
reclamation would mean shipping a separate GC for Lua values — defeating the
purpose of running on Go.

**This whole area is intentionally out of scope** for differential conformance:
the fuzzing campaigns explicitly exclude `collectgarbage`, `__gc`, and weak-table
*timing*.

## Where this lives in the source

- Finalizer registration is per-VM: `(*VM).RegisterGcFinalizer`.
- Known-divergence pins / exploration backlog:
  [`tests/official_suite_test.go`](../../tests/official_suite_test.go)
  (`knownFail` — `gc.lua:286` weak-table reclamation under Go GC), and
  [`tests/broken_weak_tables.lua`](../../tests/broken_weak_tables.lua).

## If you need deterministic cleanup

Don't rely on GC timing from Lua. Use explicit lifetimes: to-be-closed
variables (`local x <close> = ...`) for scope-bound cleanup, or an explicit
`close()`/release method on resources you own.
