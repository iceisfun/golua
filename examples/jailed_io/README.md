# Jailed IO Example

Demonstrates sandboxed file IO and OS operations using GoLua's capability-based providers.

## What it shows

- **JailedIoProvider**: Read-only file access restricted to a specific directory
- **DefaultOsProvider**: Safe OS operations (clock, time, date, getenv)
- **Filtered environment**: Restricting which environment variables Lua can access
- **Write rejection**: Write modes are denied by the jailed provider

## Run

```bash
go run ./examples/jailed_io
```

## Key concepts

### JailedIoProvider

```go
ioProvider := vm.NewJailedIoProvider("/path/to/allowed/dir")
v.SetIoProvider(ioProvider)
```

- Uses `os.DirFS` to prevent path traversal
- Only allows `"r"` and `"rb"` modes (read-only)
- Exposes `io.open`, `io.close`, `io.lines`, `io.type` to Lua

### DefaultOsProvider

```go
osProvider := vm.NewDefaultOsProvider()
v.SetOsProvider(osProvider)
```

Exposes `os.clock`, `os.time`, `os.date`, `os.difftime`, `os.getenv` to Lua.

### Filtered environment

```go
filtered := vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
})
```

Only the listed environment variables are accessible.
