# Time Example

Non-standard extension for millisecond-precision timing.

## Functions

| Function | Description |
|---|---|
| `time.now()` | Returns current time in milliseconds (integer) |
| `time.since(t)` | Returns milliseconds elapsed since `t` |

## Security

The time table is **absent by default** (`time == nil`). It only appears
when the host explicitly sets a `LuaTimeProvider` before calling `stdlib.Open()`.

## Interface

```go
type LuaTimeProvider interface {
    Now() int64
}
```

## Usage

```go
v := vm.New()
v.SetTimeProvider(vm.NewDefaultTimeProvider())
stdlib.Open(v)
```

```lua
local start = time.now()
-- ... do work ...
print(time.since(start) .. "ms")
```

## Running

```bash
go run ./examples/time
```
