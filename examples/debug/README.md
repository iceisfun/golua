# Diagnostic Debug Example

This example demonstrates the capability-gated debug provider by implementing a
custom `LuaDebugProvider` that exposes **only** read-only diagnostic functions.

The debug table is assembled from the flags returned by the provider's
`Capabilities`, so the host decides exactly which functions exist.

> **Note:** `vm.NewDefaultDebugProvider()` enables the *full* debug library
> (hooks, locals, upvalues, registry, everything). To get the restricted,
> diagnostic-only surface shown here you must implement the provider yourself
> and return only the flags you want — that is what this example does.

## Supported Diagnostic Functions

| Function | Description |
|---|---|
| `debug.traceback([message [, level]])` | Returns a stack traceback string |
| `debug.stackdepth()` | Returns the current call stack depth |
| `debug.where([level])` | Returns source file and line number |

## Explicitly Unsupported

In this example the following standard Lua debug functions are **left out**
(their capability flags are `false`, so they never appear in the `debug` table):

- `debug.sethook` / `debug.gethook` — no hook support
- `debug.getlocal` / `debug.setlocal` — no local variable access
- `debug.getupvalue` / `debug.setupvalue` — no upvalue mutation
- `debug.getinfo` — no full frame introspection
- `debug.setmetatable` — no metatable mutation via debug
- `debug.upvalueid` / `debug.upvaluejoin` — no upvalue identity

## Security

The debug table is **absent by default** (`debug == nil`). It only appears
when the host explicitly sets a `LuaDebugProvider` before calling `stdlib.Open()`.

## Interface

```go
// LuaDebugCaps has one Allow* flag per debug.* function. Only the three
// diagnostic flags are shown here; see vm/debug_provider.go for the full set
// (getinfo, getlocal, setlocal, getupvalue, sethook, getregistry, ...).
type LuaDebugCaps struct {
    AllowTraceback  bool
    AllowStackDepth bool
    AllowWhere      bool
    // ... plus the full-library flags, all false in this example
}

type LuaDebugProvider interface {
    Capabilities(ctx context.Context) LuaDebugCaps
}
```

## Usage

```go
// Custom provider returning only the diagnostic capability flags.
type diagnosticOnly struct{}

func (diagnosticOnly) Capabilities(ctx context.Context) vm.LuaDebugCaps {
    return vm.LuaDebugCaps{AllowTraceback: true, AllowStackDepth: true, AllowWhere: true}
}

v := vm.New()
if err := v.SetDebugProvider(diagnosticOnly{}); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Running

```bash
go run ./examples/debug
```
