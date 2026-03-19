# Context Cancellation Example

Demonstrates using `context.Context` to stop a runaway Lua script.

A game server runs Lua AI scripts for NPCs. When the game round ends
(after 500ms), the context is cancelled and the VM stops the script
immediately — even though it contains an infinite loop.

## Run

```bash
go run ./examples/context
```

## Output

The exact tick counts and elapsed time vary by machine, but the output shape
looks like this:

```
=== Context Cancellation Example ===

Game round started — NPC AI script running...
tick=100000 dt=0.000
tick=200000 dt=0.000
tick=300000 dt=0.000

Script stopped after 500ms
Exit reason: execution interrupted: context deadline exceeded

=== Game round over ===
```
