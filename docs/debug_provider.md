# Debug Provider

The `LuaDebugProvider` interface gates access to the `debug` library. Without a debug provider, the `debug` table is absent.

## Quick Start

```go
v := vm.New()
if err := v.SetDebugProvider(vm.NewDefaultDebugProvider()); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Interface

```go
type LuaDebugProvider interface {
    Capabilities(ctx context.Context) LuaDebugCaps
}

type LuaDebugCaps struct {
    AllowTraceback      bool
    AllowStackDepth     bool
    AllowWhere          bool
    AllowGetInfo        bool
    AllowGetUpvalue     bool
    AllowSetUpvalue     bool
    AllowUpvalueID      bool
    AllowGetLocal       bool
    AllowSetLocal       bool
    AllowGetRegistry    bool
    AllowGetMetatable   bool
    AllowSetMetatable   bool
    AllowSetHook        bool
    AllowGetHook        bool
    AllowUpvalueJoin    bool
    AllowSetCStackLimit bool
    AllowGetUserValue   bool
    AllowSetUserValue   bool
}
```

### Capabilities

Each capability independently gates one or more `debug.*` functions:

| Capability | Functions |
|-----------|-----------|
| `AllowTraceback` | `debug.traceback` |
| `AllowStackDepth` | stack depth queries |
| `AllowGetInfo` | `debug.getinfo` |
| `AllowGetLocal` | `debug.getlocal` |
| `AllowSetLocal` | `debug.setlocal` |
| `AllowGetUpvalue` | `debug.getupvalue` |
| `AllowSetUpvalue` | `debug.setupvalue` |
| `AllowUpvalueID` | `debug.upvalueid` |
| `AllowUpvalueJoin` | `debug.upvaluejoin` |
| `AllowSetHook` | `debug.sethook` |
| `AllowGetHook` | `debug.gethook` |
| `AllowGetMetatable` | `debug.getmetatable` |
| `AllowSetMetatable` | `debug.setmetatable` |
| `AllowGetRegistry` | `debug.getregistry` |

## Default Implementation

```go
provider := vm.NewDefaultDebugProvider()
```

Enables all capabilities. Use a custom provider to selectively disable dangerous operations like `debug.setlocal` or `debug.sethook` in untrusted environments.

## Custom Implementation

```go
type ReadOnlyDebug struct{}

func (d *ReadOnlyDebug) Capabilities(ctx context.Context) vm.LuaDebugCaps {
    return vm.LuaDebugCaps{
        AllowTraceback:  true,
        AllowGetInfo:    true,
        AllowGetLocal:   true,
        AllowGetUpvalue: true,
        AllowUpvalueID:  true,
        AllowGetHook:    true,
        AllowGetMetatable: true,
        // All Set* and mutation capabilities left false
    }
}
```

## Security

- Without a provider, `debug.*` does not exist
- `debug.sethook`, `debug.setlocal`, `debug.setupvalue`, and `debug.upvaluejoin` can modify running code — disable these for untrusted scripts
- `debug.getregistry` exposes VM internals — disable for sandboxed scripts
- `debug.setmetatable` can change any value's metatable — disable for safety
