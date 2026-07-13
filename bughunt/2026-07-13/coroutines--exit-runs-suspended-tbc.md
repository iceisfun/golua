# coroutines: interpreter shutdown runs __close of suspended coroutines' pending <close> vars; reference does not

At CLI exit (both falling off the end of the script and `os.exit(0, true)`
which closes the state), golua runs the pending `__close` metamethods of
coroutines that are still suspended. Reference 5.5's `lua_close` does NOT close
to-be-closed variables of non-main threads — a suspended coroutine's tbc
handlers simply never run unless coroutine.close is called.

Found by random generator (gen/g00121.lua), minimized:

## Repro

```lua
local co = coroutine.create(function()
  local t <close> = setmetatable({}, {__close=function(_,e) print("EXIT-CLOSE", tostring(e)) end})
  coroutine.yield()
end)
coroutine.resume(co)
print("script end")
```

## golua

```
script end
EXIT-CLOSE	nil
```

## lua5.5.0

```
script end
```

Same divergence with `os.exit(0, true)` replacing the last line (golua prints
EXIT-CLOSE, reference prints nothing).

Possibly intentional (golua must close suspended coroutines to reap their
backing goroutines — see wontfix/coroutine-goroutine-leak), but it is a
user-visible side-effect divergence: __close handlers observably fire at exit
in golua and never in reference. If intentional it belongs in wontfix/.

## Verification: REJECTED (WONTFIX-SCOPE)

The divergence is real and reproduces exactly as reported (verified 2026-07-13
against the scratchpad golua binary and /usr/bin/lua5.5.0). But it is already
the *documented* behavior of an existing wontfix entry, not a new bug:

- `wontfix/coroutine-goroutine-leak/README.md` states explicitly (Mitigation,
  "What" section): "`VM.Close(ctx)` reaps the goroutines of all still-suspended
  coroutines (**it closes each resume channel, so the goroutine runs its
  `<close>` handlers and returns**)."
- The implementation matches: `stdlib/coroutine.go` — `openCoroutine` registers
  `v.OnClose(func(ctx) { reapCoroutines(v, ctx) })` (line ~145);
  `reapCoroutine`'s doc comment says the goroutine "unwinds via CloseAllTBC",
  i.e. running `__close` handlers is the mechanism, identical to
  `coroutine.close()`. The CLI calls `v.Close(...)` at exit
  (`cmd/lua/main.go:205`), which is why the handler fires after "script end".
- This cannot be "fixed" to match reference: Go cannot kill a parked goroutine.
  The only way to reclaim a suspended coroutine's goroutine is to wake it and
  let it return, and unwinding a Lua frame with pending `<close>` variables
  runs their handlers. Suppressing the handlers would mean either leaking the
  goroutines (the very leak the wontfix mitigates) or unwinding a Lua stack
  while skipping its declared `<close>` semantics.
- Reference behavior for comparison: `lua_close` closes tbc variables of the
  main thread only; abandoned suspended coroutines are simply collected without
  running `__close` (thread finalization does not invoke tbc handlers). golua's
  divergence is the deliberate, documented price of goroutine-backed
  coroutines.

The report's own last paragraph anticipated this ("If intentional it belongs
in wontfix/") — it is intentional, and it is already in wontfix/. At most, the
coroutine-goroutine-leak README could gain this repro as an example of the
user-visible `__close`-at-exit side effect, but no new wontfix entry or code
change is warranted.
