# CLI: arg[-1] (interpreter name) missing from arg table

Reference `lua` places the interpreter name and any pre-script options at
negative indices of the global `arg` table (`arg[-1]` = interpreter path when
no options are given). The golua CLI leaves all negative indices nil, so the
official suite's idiom for locating the running interpreter
(`while arg[i] do i = i - 1 end; progname = arg[i+1]`) yields the *script* name
instead, and files.lua's popen/execute self-exec tests run `sh -c 'files.lua -e ...'`
and fail (files.lua:807 assertion via wrong progname).

## Minimized repro (run as `<interp> min.lua`, no script args needed)
```lua
print(arg[-1])
```

## golua (verified 2026-07-13)
```
nil
```

## lua5.5.0 (verified 2026-07-13)
```
/usr/bin/lua5.5.0
```

Fuller view (`<interp> argtest.lua one two`, `for i = -3, 2 do print(i, arg[i]) end`):
golua prints nil at -3..-1; lua5.5.0 prints the interpreter path at -1. Indices
0..2 agree on both.

Why wrong: Lua 5.5 manual §7 (Standalone): "Any arguments before the script name
(that is, the interpreter name plus its options) go to negative indices."
Reference `lua.c createargtable` stores every argv entry at index `i - script`.
golua's CLI (`cmd/lua/main.go:182-187`) only sets `arg[0]` and positives.
Impact confirmed in the official suite: `main.lua:23-27` walks negative indices
(`while arg[i] do i=i-1 end; progname = arg[i+1]`); under golua the walk stops
at i=-1 so progname = arg[0] = the *script* name, corrupting every RUN/NoRun
self-exec (files.lua popen/execute tests included).

## Verification: CONFIRMED
Reproduced on both interpreters (golua scratchpad build vs /usr/bin/lua5.5.0,
2026-07-13). Not in wontfix/ index; not GC/finalization-dependent; not an
error-message-only divergence. Fix belongs in the standalone CLI arg-table
construction, not the VM.
