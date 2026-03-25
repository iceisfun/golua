# Exit Handler

The `LuaExitHandler` interface controls `os.exit()` behavior. Without both an exit handler and an OS provider with `AllowExit`, `os.exit` is unavailable.

## Quick Start

```go
v := vm.New()
v.SetOsProvider(vm.NewDefaultOsProvider())
v.SetExitHandler(vm.NewDefaultExitHandler())
stdlib.Open(v)
```

## Interface

```go
type LuaExitHandler interface {
    Exit(ctx context.Context, code int, close bool)
}
```

### Exit

Called when Lua executes `os.exit(code, close)`. The `close` parameter indicates whether Lua requested to-be-closed variable finalization before exiting.

The handler decides what "exit" means for the host — it could stop the VM, terminate the process, or take any other action.

## Default Implementation

```go
handler := vm.NewDefaultExitHandler()
```

Panics with `*vm.LuaExitError`, a sentinel error that propagates through `ProtectedCall` and is **not** caught by `pcall`/`xpcall`. This stops VM execution and lets the Go host recover the exit code.

```go
type LuaExitError struct {
    Code int
}
```

### Recovering the exit code

```go
results, err := v.ProtectedCall(fn, nil)
if err != nil {
    var exitErr *vm.LuaExitError
    if errors.As(err, &exitErr) {
        fmt.Printf("script exited with code %d\n", exitErr.Code)
    }
}
```

## Custom Implementation

```go
type LogOnlyExit struct{}

func (e *LogOnlyExit) Exit(ctx context.Context, code int, close bool) {
    log.Printf("lua requested exit with code %d", code)
    // Do not panic — script continues running
}
```

## Security

- Without a handler, `os.exit` does not exist
- The default handler stops the VM via panic; custom handlers can choose any behavior
- `LuaExitError` is not catchable by Lua's `pcall`/`xpcall`
