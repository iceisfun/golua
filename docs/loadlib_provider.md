# LoadLib Provider

The `LuaLoadLibProvider` interface controls `package.loadlib()` behavior. Without a loadlib provider, `package.loadlib` returns Lua's standard "absent" failure triple.

## Quick Start

```go
v := vm.New()
if err := v.SetLoadLibProvider(&MyLoadLibProvider{}); err != nil {
    log.Fatal(err)
}
stdlib.Open(v)
```

## Interface

```go
type LuaLoadLibProvider interface {
    LoadLib(ctx context.Context, path, init string, caller *LuaCallerContext) (loader NativeFunc, errmsg string, where string)
}
```

### LoadLib

Resolves `(path, init)` to a callable loader function. The `path` and `init` arguments match Lua 5.4's `package.loadlib` signature. The host decides how to interpret them — they need not refer to actual `.so`/`.dll` files.

- **On success**: return `(NativeFunc, "", "")`
- **On failure**: return `(nil, errmsg, where)` where `where` is `"open"`, `"init"`, or `"absent"`

The `caller` argument provides the requesting script's name, VM ID, and call depth.

## Example

```go
type MyLoadLibProvider struct{}

func (p *MyLoadLibProvider) LoadLib(ctx context.Context, path, init string, caller *vm.LuaCallerContext) (vm.NativeFunc, string, string) {
    if path == "mylib" && init == "luaopen_mylib" {
        return func(v *vm.VM) int {
            lib := vm.NewEmptyTable()
            lib.SetString("greet", vm.NewNativeFunc(func(v *vm.VM) int {
                v.Set(0, vm.NewString("hello from Go"))
                return 1
            }))
            v.Set(0, vm.NewTable(lib))
            return 1
        }, "", ""
    }
    return nil, fmt.Sprintf("%s: module not available", path), "open"
}
```

## Background

Standard C Lua modules (`.so`/`.dll`) are compiled against the PUC-Rio C API (`lua_State*`, `lua_push*`, etc.) and cannot be loaded directly into GoLua. This provider lets the host implement its own native module strategy:

- Map module names to Go-implemented bindings
- Use cgo to bridge platform-specific libraries
- Pre-register native functions under known names

## Security

- Without a provider, `package.loadlib` always returns the "absent" failure
- The host has full control over which modules are loadable
- `require` loads Lua modules via `LuaCodeProvider`, not this provider
