# Chan Provider

The `LuaChanProvider` interface controls the `chan` module for Go-to-Lua message passing. Without a chan provider, the `chan` table is absent (`chan == nil`).

## Quick Start

```go
provider := vm.NewDefaultChanProvider()
events := provider.NewChannel(context.Background(), 0) // unbuffered

v := vm.New()
v.SetChanProvider(provider)
stdlib.Open(v)

v.SetGlobal("events", stdlib.WrapChannel(v, events))
```

Or use the convenience function:

```go
v := vm.New()
stdlib.ProvideChan(v) // sets provider and opens module
```

## Interface

```go
type LuaChanProvider interface {
    NewChannel(ctx context.Context, size int) *LuaChannel
    Capabilities(ctx context.Context) LuaChanCaps
}

type LuaChanCaps struct {
    AllowSend    bool
    AllowRecv    bool
    AllowClose   bool
    AllowSelect  bool
    AllowTrySend bool
    AllowTryRecv bool
}
```

### LuaChannel

Channels are created by the provider and can be used from both Go and Lua:

```go
// Go side
ch := provider.NewChannel(0)
ch.Send(ctx, vm.NewString("hello"))
val, ok, err := ch.Recv(ctx)
ch.Close()

// Non-blocking
sent := ch.TrySend(vm.NewString("data"))
val, ok, received := ch.TryRecv()
```

## Lua API

| Function | Description |
|----------|-------------|
| `chan.make(size?)` | Create a new channel (0 = unbuffered) |
| `chan.select(ch1, ..., timeout?)` | Receive from any ready channel; returns `idx, val, ok` |
| `ch:send(val)` | Blocking send |
| `ch:recv()` | Blocking receive; returns `val, ok` |
| `ch:close()` | Close the channel |
| `ch:try_send(val)` | Non-blocking send; returns `bool` |
| `ch:try_recv()` | Non-blocking receive; returns `val, ok, received` |

`chan.select` returns a 1-based index of the channel that fired, or `0` on timeout. Blocking operations respect context cancellation.

## Default Implementation

```go
provider := vm.NewDefaultChanProvider()
```

All capabilities enabled. Channels use Go's native `chan` type. Each channel gets a unique auto-incrementing ID.

## Security

- Without a provider, `chan` does not exist
- Capabilities independently gate each operation
- Channels from different providers are rejected by `chan.select` (VM boundary safety)
- No goroutines or shared memory are exposed to Lua
