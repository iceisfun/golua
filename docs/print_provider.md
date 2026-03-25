# Print Provider

The `LuaPrintProvider` interface routes `print()` and `warn()` output through host-defined handlers. Without a print provider, `print()` writes to stdout (or the `WithCaptureOutput` buffer) and `warn()` writes to stderr.

## Quick Start

```go
type Logger struct{ Name string }

func (l *Logger) Print(ctx context.Context, msg string) {
    log.Printf("[%s] %s", l.Name, msg)
}
func (l *Logger) Warn(ctx context.Context, msg string) {
    log.Printf("[%s] WARN: %s", l.Name, msg)
}

v := vm.New()
v.SetPrintProvider(&Logger{Name: "inventory.lua"})
stdlib.Open(v)
```

## Interface

```go
type LuaPrintProvider interface {
    Print(ctx context.Context, msg string)
    Warn(ctx context.Context, msg string)
}
```

### Print

Called by Lua's `print()`. The `msg` argument contains all arguments tab-joined into a single string. The provider is responsible for adding a newline if desired.

### Warn

Called by Lua's `warn()`. The `msg` argument includes the `"Lua warning: "` prefix. The provider is responsible for adding a newline if desired.

## Default Implementation

```go
provider := vm.NewDefaultPrintProvider()
```

Writes print output to stdout with `fmt.Println()` and warn output to stderr with `fmt.Fprintln()`.

## Behavior Notes

- `print()` and `warn()` are always available (they do not require a provider)
- Without a provider, the VM uses built-in stdout/stderr behavior
- The `warn("@on")`/`warn("@off")` control flag is per-VM — disabling warnings in one VM does not affect others
- When a `LuaPrintProvider` is set, output routes to it exclusively; output capture (`vm.WithCaptureOutput(true)`) only applies when no provider is set

## Use Cases

- Route Lua output through structured logging (zap, slog, logrus)
- Capture output per-VM for multi-tenant environments
- Silence output in background workers
- Redirect warnings to monitoring systems
