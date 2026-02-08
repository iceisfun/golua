# Lua-to-Go Channel Example

Demonstrates Lua sending messages to Go via a buffered channel.

## What it shows

- Creating a buffered channel so Lua won't block
- Lua producing messages with `ch:send(val)` and closing with `ch:close()`
- Go reading messages from the channel after `v.Run()` completes
- Using `LuaChannel.Recv()` on the Go side to drain the channel

## Run

```bash
go run ./examples/chan/lua_to_go
```
