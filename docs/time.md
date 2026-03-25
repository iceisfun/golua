# Time Provider

The `LuaTimeProvider` interface controls the `time` module for millisecond-precision timing. Without a time provider, the `time` table is absent.

## Quick Start

```go
v := vm.New()
v.SetTimeProvider(vm.NewDefaultTimeProvider())
stdlib.Open(v)
```

## Interface

```go
type LuaTimeProvider interface {
    Now(ctx context.Context) int64
    Tick(ctx context.Context, key string, ms int64) bool
    Once(ctx context.Context, key string) bool
}
```

### Now

Returns current time in milliseconds.

### Tick

Returns `true` once per `ms` milliseconds for a given key. First call for a key always returns `true`. Used for periodic triggers without coroutines or timers.

### Once

Returns `true` on the first call for a given key, `false` on all subsequent calls. Used for one-time initialization guards.

## Lua API

| Function | Description |
|----------|-------------|
| `time.now()` | Current time in milliseconds (integer) |
| `time.since(t)` | Milliseconds elapsed since `t` |
| `time.tick([name,] ms)` | Returns `true` once per `ms` interval |
| `time.once([name])` | Returns `true` on first call for key |

When `name` is omitted, `time.tick` and `time.once` auto-key by callsite — the VM inspects the calling function's source file and line number (`source:line`) so each call location gets independent state. Pass an explicit `name` to share state across call locations.

```lua
local start = time.now()
-- ... work ...
print(time.since(start) .. "ms")

for i = 1, math.huge do
    if time.tick(1000) then print("once per second") end
end

if time.once() then load_resources() end
```

## Default Implementation

```go
provider := vm.NewDefaultTimeProvider()
```

Uses `time.Now().UnixMilli()` for timestamps. Tick and once state is stored in mutex-protected maps with limits: max 10,000 distinct keys, keys truncated to 512 bytes. New keys beyond the limit are silently ignored (tick returns `false`, once returns `false`).

## Security

- Without a provider, `time` does not exist
- `time.tick` and `time.once` are GoLua extensions (not part of standard Lua)
- Key limits prevent memory exhaustion from unbounded key creation
