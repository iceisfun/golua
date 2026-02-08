# Multi-Producer chan.select Example

Demonstrates multiple Go producers sending to separate channels, with Lua multiplexing them using `chan.select`.

## What it shows

- Multiple unbuffered channels for synchronous handoff
- A "done" channel as a stop signal (fan-in pattern)
- `chan.select(ch1, ch2, ch3, done)` to receive from whichever channel has data
- Index-based dispatch: `chan.select` returns the 1-based index of the channel that fired
- Sequential producers for deterministic output

## Run

```bash
go run ./examples/chan/multi_go_to_lua
```
