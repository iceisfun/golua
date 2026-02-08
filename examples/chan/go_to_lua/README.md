# Go-to-Lua Channel Example

Demonstrates a Go goroutine pushing events to a running Lua script via an unbuffered channel.

## What it shows

- Creating a `DefaultChanProvider` and a channel from Go
- Wrapping the channel with `stdlib.WrapChannel` to pass it into Lua
- Lua consuming messages in a `while true` loop with `ch:recv()`
- Detecting channel closure via the `ok` return value (`false` = closed and drained)
- Synchronous handoff: Go's `Send` blocks until Lua reads

## Run

```bash
go run ./examples/chan/go_to_lua
```
