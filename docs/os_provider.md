# OS Provider

The `LuaOsProvider` interface controls `os.clock`, `os.time`, `os.date`, `os.difftime`, `os.getenv`, and `os.setlocale`. Without an OS provider, the `os` table is absent.

Additional `os.*` functions require their own providers: `os.execute` needs `LuaExecProvider`, `os.exit` needs `LuaExitHandler`, and `os.remove`/`os.rename`/`os.tmpname` need `LuaIoProvider`.

## Quick Start

```go
v := vm.New()
v.SetOsProvider(vm.NewDefaultOsProvider())
stdlib.Open(v)
```

## Interface

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

type LuaTimeInput struct {
    Year     int
    Month    int   // 1-12
    Day      int   // 1-31
    Hour     int   // default 12 when omitted
    Min      int   // default 0
    Sec      int   // default 0
    HasIsDST bool
    IsDST    bool
}

type LuaDateTime struct {
    Year   int
    Month  int   // 1-12
    Day    int   // 1-31
    Hour   int   // 0-23
    Min    int   // 0-59
    Sec    int   // 0-60
    Wday   int   // 1-7 (Sunday=1)
    Yday   int   // 1-366
    IsDST  bool
    HasDST bool
}
```

### Capabilities

Each capability independently gates its corresponding `os.*` function:

| Capability | Functions |
|-----------|-----------|
| `AllowTime` | `os.clock()`, `os.time()`, `os.difftime()` |
| `AllowDate` | `os.date()` |
| `AllowGetenv` | `os.getenv()` |
| `AllowTmpName` | `os.tmpname()` (also requires `LuaIoProvider`) |
| `AllowRemove` | `os.remove()` (also requires `LuaIoProvider`) |
| `AllowRename` | `os.rename()` (also requires `LuaIoProvider`) |
| `AllowExecute` | `os.execute()` (also requires `LuaExecProvider`) |
| `AllowExit` | `os.exit()` (also requires `LuaExitHandler`) |

## Default Implementations

### DefaultOsProvider

```go
provider := vm.NewDefaultOsProvider()
```

All capabilities enabled. Uses `time.Now()` for time functions, `os.Getenv` for environment variables, and Go's `time` package for date formatting with full strftime support.

`os.clock()` returns elapsed CPU time since the provider was created. `os.setlocale` only supports the `"C"` locale (Go has no native locale support).

Date formatting supports all standard strftime specifiers: `%Y`, `%m`, `%d`, `%H`, `%M`, `%S`, `%A`, `%a`, `%B`, `%b`, `%c`, `%x`, `%X`, `%Z`, `%z`, and more. Use `!` prefix for UTC (e.g., `os.date("!%Y-%m-%d")`).

### FilteredOsProvider

```go
provider := vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
})
```

Same as `DefaultOsProvider` but with a filter function for `os.getenv`. Environment variables not approved by the filter return `nil`.

## Security

- Without a provider, `os.*` does not exist
- `FilteredOsProvider` restricts which environment variables are visible
- `os.execute` and `os.exit` require additional providers beyond `LuaOsProvider`
- Year range validation prevents overflow in date calculations
