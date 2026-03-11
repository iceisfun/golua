# Time Example

Non-standard extension for millisecond-precision timing.

## Functions

| Function | Description |
|---|---|
| `time.now()` | Returns current time in milliseconds (integer) |
| `time.since(t)` | Returns milliseconds elapsed since `t` |
| `time.tick([name,] ms)` | Returns `true` once per `ms` interval, `false` otherwise |
| `time.once([name])` | Returns `true` the first time a key is seen, then `false` |

### time.tick

Designed for periodic logic inside hot loops. Returns `true` at most once
per `ms` milliseconds, `false` on all other calls.

```lua
-- auto-keyed by callsite (source:line)
for i = 1, math.huge do
    if time.tick(1000) then print("once per second") end
    if time.tick(5000) then print("once per 5 seconds") end
end

-- explicit name (shares state across callsites)
if time.tick("heartbeat", 1000) then send_heartbeat() end
```

When `name` is omitted, each callsite gets its own independent timer
automatically via the source location.

### time.once

`time.once` is useful for one-time initialization inside hot loops or update callbacks.

```lua
for i = 1, 5 do
    if time.once() then
        print("run setup exactly once")
    end
end

if time.once("cache:init") then
    warm_cache()
end
```

## Security

The time table is **absent by default** (`time == nil`). It only appears
when the host explicitly sets a `LuaTimeProvider` before calling `stdlib.Open()`.

## Interface

```go
type LuaTimeProvider interface {
    Now() int64
    Tick(key string, ms int64) bool
}
```

## Usage

```go
v := vm.New()
v.SetTimeProvider(vm.NewDefaultTimeProvider())
stdlib.Open(v)
```

## Running

```bash
go run ./examples/time
```
