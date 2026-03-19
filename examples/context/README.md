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

```
=== Context Cancellation Example ===

Game round started — NPC AI script running...
NPC thinking... tick 1000000
NPC thinking... tick 2000000
NPC thinking... tick 3000000

Script stopped after 501ms
Exit reason: execution interrupted: context deadline exceeded

=== Game round over ===
```
