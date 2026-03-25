# Exec Provider

The `LuaExecProvider` interface controls `os.execute()` command execution. This is separate from the `exec` module (which uses `LuaProcessProvider`). Without both an exec provider and an OS provider with `AllowExecute`, `os.execute` is unavailable.

## Quick Start

```go
v := vm.New()
if err := v.SetOsProvider(vm.NewDefaultOsProvider()); err != nil {
    log.Fatal(err)
}
if err := v.SetExecProvider(vm.NewDefaultExecProvider()); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Interface

```go
type LuaExecProvider interface {
    Execute(ctx context.Context, command string) (ok bool, exitType string, exitCode int)
}
```

### Execute

Runs a shell command and returns three values matching Lua 5.4's `os.execute`:

- `ok` — true if the command exited with status 0
- `exitType` — `"exit"` for normal termination, `"signal"` for signal-based exit
- `exitCode` — the exit status code or signal number

When called with an empty command, returns `(true, "exit", 0)` to indicate a shell is available.

## Default Implementation

```go
provider := vm.NewDefaultExecProvider()
```

Runs commands via `sh -c <command>` using `exec.CommandContext`. Stdout and stderr are forwarded to the host process's stdout/stderr (not captured). Command-not-found returns `(false, "exit", 127)`.

## Comparison with exec Module

| Feature | `os.execute` | `exec` module |
|---------|-------------|---------------|
| Provider | `LuaExecProvider` | `LuaProcessProvider` |
| Output | Forwarded to host | Captured or streamed |
| Stdin | Not available | Writable pipe |
| Async | No | `exec.spawn` |
| Shell mode | Always | `exec.run_shell` only |

For modern process control with captured output, streaming I/O, and spawning, use the [`exec` module](exec.md) instead.

## Security

- Requires both `LuaOsProvider` (with `AllowExecute`) and `LuaExecProvider`
- Neither provider is set by default
- Custom implementations can restrict allowed commands, sanitize arguments, or log execution
