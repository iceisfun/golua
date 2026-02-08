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

## Interfaces

The `io` and `os` libraries are gated behind provider interfaces. If no provider is set, the corresponding library is not registered at all.

This is useful for applications like games or embedded systems that may need a virtual filesystem, a simulated clock, or other non-standard implementations.

### LuaIoProvider

Controls how Lua opens and reads files. Implement this to back `io.open`, `io.close`, `io.lines`, and `io.type`.

```go
type LuaIoProvider interface {
    Open(name string, mode string) (LuaFile, error)
    Capabilities() LuaIoCaps
}

type LuaFile interface {
    Read(format string) (string, error)  // "a", "l", "L", "n"
    ReadBytes(n int) (string, error)
    Close() error
    IsClosed() bool
}

type LuaIoCaps struct {
    AllowRead  bool
    AllowWrite bool
}
```

Examples: real filesystem, in-memory virtual filesystem, archive-backed assets, network-mounted storage.

### LuaOsProvider

Controls what system information Lua can access. Implement this to back `os.clock`, `os.time`, `os.date`, `os.difftime`, and `os.getenv`.

```go
type LuaOsProvider interface {
    Clock() float64
    Time(dateTable map[string]int) (int64, error)
    Date(format string, timestamp int64) (string, error)
    DateTable(timestamp int64) map[string]int
    Getenv(name string) (string, bool)
    Capabilities() LuaOsCaps
}

type LuaOsCaps struct {
    AllowTime   bool
    AllowDate   bool
    AllowGetenv bool
}
```

Examples: simulated game clock, deterministic replay timestamps, restricted environment.

## Included implementations

### JailedIoProvider

```go
ioProvider := vm.NewJailedIoProvider("/path/to/allowed/dir")
v.SetIoProvider(ioProvider)
```

- Uses `os.DirFS` to prevent path traversal
- Only allows `"r"` and `"rb"` modes (read-only)

### DefaultOsProvider

```go
osProvider := vm.NewDefaultOsProvider()
v.SetOsProvider(osProvider)
```

Delegates to the real system clock and environment.

### Filtered environment

```go
filtered := vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
})
```

Only the listed environment variables are accessible.
