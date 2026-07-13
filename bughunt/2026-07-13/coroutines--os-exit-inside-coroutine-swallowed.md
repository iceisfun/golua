# coroutines: os.exit() called inside a coroutine does not exit the process — it becomes a catchable resume error

`os.exit(n)` in the main thread works (process exits with n, and
`pcall(os.exit, 3)` correctly does NOT catch it). But inside a coroutine, the
special exit panic is intercepted by the resume machinery and surfaces as an
ordinary coroutine error: `coroutine.resume` returns
`false, "os.exit(3)"`, the coroutine is marked dead, and the host process keeps
running. Reference exits the process immediately with the requested status.

## Repro

```lua
local co = coroutine.create(function()
  os.exit(3)
end)
print("resume:", coroutine.resume(co))
print("status:", coroutine.status(co))
print("still alive")
```

## golua

```
resume:	false	os.exit(3)
status:	dead
still alive
(exit code 0)
```

## lua5.5.0

```
(no output; exit code 3)
```

## Variant: coroutine.wrap

```lua
coroutine.wrap(function() os.exit(false) end)()
print("alive after wrap-exit")
```

golua propagates it as a Lua error out of the wrap call — CLI prints
`golua: FILE:1: os.exit(1)` plus a traceback to stderr and exits 1 (the "1"
matching reference's exit(false) status is coincidental: it's the CLI's
generic error exit code). Reference exits silently with status 1 and no error
output. In the wrap variant a `pcall` around the wrap call would swallow the
exit entirely.

Impact: scripts using os.exit from worker coroutines keep running; the internal
sentinel string "os.exit(N)" leaks as a user-visible error message; pcall'd
coroutine code can cancel an exit request. The exit panic must propagate
through (or be re-thrown across) the coroutine resume boundary like it does
through pcall in the main thread.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced with the current binary against `/usr/bin/lua5.5.0`.

Minimized repro (2 lines):

```lua
coroutine.resume(coroutine.create(function() os.exit(3) end))
print("unreachable")
```

- golua: prints `unreachable`, exit code 0. `coroutine.resume` returns
  `false, "os.exit(3)"` (verified by printing the resume results) — fully
  catchable, coroutine marked dead.
- lua5.5.0: no output, exit code 3.

Controls verified:
- Top-level `os.exit(3)`: both exit 3 (golua's exit path itself works).
- `pcall(function() os.exit(3) end)`: both exit 3 — golua correctly propagates
  the `*vm.LuaExitError` sentinel through pcall (`vm/vm.go:422-423`), so the
  divergence is specific to the coroutine boundary.
- `coroutine.wrap` variant reproduced as described: golua exits 1 with an
  ordinary Lua error + traceback (catchable by an enclosing pcall); reference
  exits 3 silently.

Root cause: `runCoroutine`'s deferred recover in `stdlib/coroutine.go`
(~line 430) catches the panic and stores any `error` — including
`*vm.LuaExitError` — into `co.err`, so resume reports it as a normal coroutine
error. This contradicts the sentinel's documented contract in
`vm/exit_handler.go` ("recognized by ProtectedCall and propagated without
being caught by pcall/xpcall"). Fix direction: detect `*vm.LuaExitError`
(and preserve it through `CallCoroutine`'s error return) and re-panic it on
the RESUMER's goroutine in `coResume`/`coWrap`, so it propagates up through
the resumer's ProtectedCall chain exactly as in the main thread. (The panic
happens on the coroutine's goroutine, so it must be hand-carried across the
channel; re-panicking resumer-side keeps the CLI's existing
`LuaExitError` recover in `cmd/lua/main.go:189-194` working.)

Scope checks: not in `wontfix/` index (`coroutine-goroutine-leak` covers
abandoned suspended coroutines, unrelated); no GC/finalizer dependence; not
error-message-only (control flow and process exit status differ). Reference
semantics are unambiguous: `os.exit` calls C `exit()`, terminating the host
process regardless of which coroutine is running.
