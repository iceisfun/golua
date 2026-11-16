# debug-debug-console

## What

`debug.debug()` is absent: `debug.debug` is `nil` in golua. In reference Lua
it enters an interactive console that reads lines from the process's C
`stdin`, runs each one as a chunk (printing errors to `stderr`), and returns
when a line reads `cont`.

```lua
print(type(debug.debug))   -- golua: nil   reference: function
```

Every other `debug` library function is provided, each behind its
`DebugProvider` capability.

## Why this won't change

golua is an embedded interpreter. A host decides what a script can read and
write through its providers, and a script often has no terminal behind it at
all. A console that blocks on the process's raw `stdin` and runs whatever is
typed there would bypass that design, and a version reading the IO provider's
stdin would hand interactive code execution to whatever is connected to that
stream. Neither is something a sandboxed host wants by default. The console
is also purely interactive: no program depends on its result, so its absence
does not change how any non-interactive script behaves.

A host that wants such a console can register its own `debug.debug` as a
native function (or write one in Lua with `io.read` and `load`) and decide
where its input comes from.

## Where this lives in the source

- `stdlib/debug.go` (`openDebug`) registers the `debug` table, one function
  per `DebugProvider` capability; there is no entry for `debug`.
