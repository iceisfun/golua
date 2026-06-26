# coroutine-goroutine-leak

## What

golua implements each coroutine as a goroutine. A **suspended** coroutine is a
goroutine parked on its resume channel. If a suspended coroutine is **abandoned**
— never resumed to completion and never `coroutine.close()`'d — its goroutine
cannot be reclaimed *while the VM is live*, because Go cannot kill a parked
goroutine. Reference Lua simply collects an abandoned suspended coroutine.

**Mitigation built in:** `VM.Close(ctx)` reaps the goroutines of all
still-suspended coroutines (it closes each resume channel, so the goroutine runs
its `<close>` handlers and returns). So the leak is bounded to the **VM's
lifetime**, not the process's — the standard `v := vm.New(); defer v.Close(ctx)`
embedding pattern reclaims abandoned suspended coroutines. What remains
unrecoverable is only a *single, never-closed VM* that keeps abandoning
suspended coroutines forever; there, close or complete them explicitly.

```lua
for i = 1, 1000 do
  local co = coroutine.create(function() for j = 1, 100 do coroutine.yield(j) end end)
  coroutine.resume(co)   -- suspended, then dropped -> +1 goroutine, never reclaimed
end
```

Coroutines that reach **dead** (run to completion or error) and coroutines that
are explicitly **`coroutine.close()`'d** are fully reaped — both the goroutine
and the memory. Only the *abandoned suspended* case leaks.

## Why the residue won't fully go away

Go provides no way to forcibly terminate a goroutine; a goroutine must return on
its own. A suspended coroutine's goroutine is blocked receiving on its resume
channel, so it can only exit if something *wakes* it with a terminate signal —
which `coroutine.close()` does, `VM.Close()` now does for all suspended
coroutines, and the dead path does implicitly. The only case left unrecoverable
is a *single, long-lived VM that is never `Close`d* and keeps abandoning
suspended coroutines: there is no event at which to signal them. That is
bounded, embedder-controllable, and easily avoided.

## Mitigation

1. **`VM.Close(ctx)`** reaps the goroutines of all still-suspended coroutines.
   Use the standard `v := vm.New(); defer v.Close(ctx)` lifecycle and abandoned
   suspended coroutines are reclaimed when the VM is closed.
2. Within a long-lived VM, ensure each coroutine is run to completion or
   `coroutine.close()`'d (the latter is also what Lua 5.4+ to-be-closed
   semantics encourage). Iterators built with `coroutine.wrap` driven to
   exhaustion do not leak.

## Where this lives in the source

- Coroutine goroutine: `stdlib/coroutine.go` — `go runCoroutine(co)`; the
  goroutine parks on `co.resumeCh`. `coroutine.close` / completion close
  `co.doneCh` and let `runCoroutine` return.
- `VM.Close` reaping: `stdlib/coroutine.go` — `reapCoroutines` / `reapCoroutine`,
  registered via `v.OnClose` in `openCoroutine`.
- Tests: `TestCoroutineGoroutineLifecycle` (reaping paths + the leak) and
  `TestVMCloseReapsCoroutines` (Close reclaims suspended coroutines) in
  `coroutine_lifecycle_test.go`.
