# coroutine-goroutine-leak

## What

golua implements each coroutine as a goroutine. A **suspended** coroutine is a
goroutine parked on its resume channel. If a suspended coroutine is **abandoned**
— never resumed to completion and never `coroutine.close()`'d — its goroutine
leaks for the lifetime of the process. Reference Lua simply collects an abandoned
suspended coroutine.

```lua
for i = 1, 1000 do
  local co = coroutine.create(function() for j = 1, 100 do coroutine.yield(j) end end)
  coroutine.resume(co)   -- suspended, then dropped -> +1 goroutine, never reclaimed
end
```

Coroutines that reach **dead** (run to completion or error) and coroutines that
are explicitly **`coroutine.close()`'d** are fully reaped — both the goroutine
and the memory. Only the *abandoned suspended* case leaks.

## Why this won't change (without a design change)

Go provides no way to forcibly terminate a goroutine; a goroutine must return on
its own. A suspended coroutine's goroutine is blocked receiving on its resume
channel, so it can only exit if something *wakes* it with a terminate signal —
which `coroutine.close()` does, and which the dead path does implicitly. An
abandoned coroutine is, by definition, never signalled. Reaping it would require
a finalizer on the thread handle that wakes-and-terminates the parked goroutine;
that machinery (finalizer + a die path through the coroutine's channel protocol,
careful to not corrupt VM state) is a non-trivial design change, and the
behavior is bounded and easily avoided.

## Mitigation

In long-lived embeddings that create many coroutines, ensure each is either run
to completion or `coroutine.close()`'d (the latter is also what Lua 5.4+ to-be-
closed semantics encourage). Iterators built with `coroutine.wrap` that are
driven to exhaustion do not leak.

## Where this lives in the source

- Coroutine goroutine: `stdlib/coroutine.go` — `go runCoroutine(co)`; the
  goroutine parks on `co.resumeCh`. `coroutine.close` / completion close
  `co.doneCh` and let `runCoroutine` return.
- Characterization test pinning the supported (reaping) paths and recording the
  abandoned-leak: `TestCoroutineGoroutineLifecycle` in
  `coroutine_lifecycle_test.go`.
