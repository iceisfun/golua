# Diagnostic Debug Example

This example demonstrates the capability-gated diagnostic debug provider.

**This is NOT the standard Lua debug library.** Only read-only diagnostic
functions are exposed.

## Supported Diagnostic Functions

| Function | Description |
|---|---|
| `debug.traceback([message [, level]])` | Returns a stack traceback string |
| `debug.stackdepth()` | Returns the current call stack depth |
| `debug.where([level])` | Returns source file and line number |

## Explicitly Unsupported

The following standard Lua debug functions are **intentionally absent**:

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
type LuaDebugCaps struct {
    AllowTraceback  bool
    AllowStackDepth bool
    AllowWhere      bool
}

type LuaDebugProvider interface {
    Capabilities() LuaDebugCaps
}
```

## Usage

```go
v := vm.New()
v.SetDebugProvider(vm.NewDefaultDebugProvider())
stdlib.Open(v)
```

## Running

```bash
go run ./examples/debug
```
