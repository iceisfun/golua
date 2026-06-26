# Jailed IO Example

Demonstrates sandboxed file IO and OS operations using GoLua's capability-based providers.

## What it shows

- **JailedIoProvider**: Read-only file access restricted to a specific directory
- **DefaultOsProvider**: Safe OS operations (clock, time, date, getenv, setlocale)
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
    Open(ctx context.Context, name, mode string) (LuaFile, error)
    Capabilities(ctx context.Context) LuaIoCaps
    Stdin(ctx context.Context) LuaFile
    Stdout(ctx context.Context) LuaFile
    Stderr(ctx context.Context) LuaFile
    TmpName(ctx context.Context) (string, error)
    TmpFile(ctx context.Context) (LuaFile, error)
    Remove(ctx context.Context, name string) error
    Rename(ctx context.Context, oldname, newname string) error
}

type LuaFile interface {
    Read(ctx context.Context, format string) (string, error)  // "a", "l", "L", "n"
    ReadBytes(ctx context.Context, n int) (string, error)
    Write(ctx context.Context, data string) error
    Seek(ctx context.Context, whence string, offset int64) (int64, error)
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
    IsClosed() bool
    // ... plus SetVBuf; see vm/io_provider.go for the full interface
}

type LuaIoCaps struct {
    AllowRead  bool
    AllowWrite bool
}
```

Examples: real filesystem, in-memory virtual filesystem, archive-backed assets, network-mounted storage.

### LuaOsProvider

Controls what system information Lua can access. Implement this to back `os.clock`, `os.time`, `os.date`, `os.difftime`, `os.getenv`, and `os.setlocale`.

```go
type LuaOsProvider interface {
    Clock(ctx context.Context) float64
    Time(ctx context.Context, dateTable *LuaTimeInput) (int64, *LuaDateTime, error)
    Date(ctx context.Context, format string, timestamp int64) (string, error)
    DateTable(ctx context.Context, timestamp int64, utc bool) *LuaDateTime
    Getenv(ctx context.Context, name string) (string, bool)
    SetLocale(ctx context.Context, locale, category string) (string, bool)
    Capabilities(ctx context.Context) LuaOsCaps
}

type LuaOsCaps struct {
    AllowTime    bool
    AllowDate    bool
    AllowGetenv  bool
    AllowTmpName bool
    AllowRemove  bool
    AllowExecute bool
    AllowExit    bool
    AllowRename  bool
}
```

Examples: simulated game clock, deterministic replay timestamps, restricted environment.

## Included implementations

### JailedIoProvider

```go
ioProvider := vm.NewJailedIoProvider("/path/to/allowed/dir")
if err := v.SetIoProvider(ioProvider); err != nil {
    log.Fatal(err)
}
```

- Uses `os.DirFS` to prevent path traversal
- Only allows `"r"` and `"rb"` modes (read-only)

### DefaultOsProvider

```go
osProvider := vm.NewDefaultOsProvider()
if err := v.SetOsProvider(osProvider); err != nil {
    log.Fatal(err)
}
```

Delegates to the real system clock and environment.

### Filtered environment

```go
filtered := vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
})
```

Only the listed environment variables are accessible.
