# Exposing Go-Backed Objects to Lua

This example demonstrates the canonical pattern for giving Lua access to
Go-owned state through an explicit adapter layer, paired with a Lua companion
module that adds behavior.

## Run

```sh
go run ./examples/expose_object
```

---

## Ownership Model

**Go owns state and lifetime.** The `Enemy` struct in `enemy.go` holds all
mutable fields (health, position, alive/dead). Go enforces invariants: health
is clamped, dead entities ignore mutations, and no field is directly exposed.

**Lua owns behavior and policy.** The `enemy.lua` companion module wraps the
Go-provided table and adds AI decisions (flee, heal, advance), status
formatting, and turn logic. Lua never touches Go memory directly.

The boundary is a table of closures. Each closure captures the Go `*Enemy`
pointer and delegates to a specific Go method. Mutations flow through Go
methods; policy flows through Lua functions.

---

## Directory Layout

```
examples/expose_object/
├── enemy.go          # Pure Go core (state + invariants)
├── lua_enemy.go      # Go → Lua adapter (closure table)
├── enemy.lua         # Lua companion module (behavior layer)
├── main.go           # Wiring + execution
└── README.md         # This file
```

---

## Go API Surface

`EnemyToLua(e *Enemy) *vm.Table` converts an `*Enemy` into a Lua table with
these functions:

| Function       | Signature (Lua)       | Returns           | Description                       |
|----------------|-----------------------|-------------------|-----------------------------------|
| `name()`       | `() -> string`        | Enemy name        | Read-only accessor                |
| `health()`     | `() -> integer`       | Current HP        | Read-only accessor                |
| `max_hp()`     | `() -> integer`       | Maximum HP        | Read-only accessor                |
| `position()`   | `() -> number, number`| x, y coordinates  | Read-only accessor                |
| `is_alive()`   | `() -> boolean`       | Alive state       | Read-only accessor                |
| `take_damage(n)` | `(integer) -> ()`   | Nothing           | Reduces HP, may kill              |
| `heal(n)`      | `(integer) -> ()`     | Nothing           | Restores HP, clamped to max       |
| `move_to(x,y)` | `(number, number) -> ()` | Nothing        | Sets position, ignored if dead    |

All functions are plain function calls (not methods). They capture the Go
pointer internally, so no `self` argument is needed.

---

## Lua Usage

```lua
local Enemy = dofile("enemy.lua")

-- Wrap a Go-provided table into a Lua enemy
local goblin = Enemy.wrap(goblin_core)

-- Accessors (delegated to Go)
print(goblin:name())            --> "Goblin"
print(goblin:health())          --> 50
print(goblin:is_alive())        --> true

-- Behavior (defined in Lua)
print(goblin:status())          --> "Goblin: 50/50 HP at (10.0, 5.0)"
print(goblin:should_flee())     --> false

-- Mutate through Go adapter
goblin_core.take_damage(35)

-- Policy reflects new state
print(goblin:should_heal())     --> true
local action = goblin:take_turn(0, 0)
print(action)                   --> "Goblin heals for 10"
```

---

## Safety Guarantees

- **Dead entities are safe.** Calling `take_damage`, `heal`, or `move_to` on a
  dead enemy is a no-op. Go enforces this; Lua cannot bypass it.

- **No panics escape into Lua.** All Go methods check preconditions (alive
  state, non-negative amounts) before mutating. Invalid inputs are silently
  ignored, not errors.

- **No shared mutable state is exposed.** Lua receives closures, not struct
  pointers. There is no way to read or write a Go field without going through
  an explicit method.

- **Go-side verification.** After Lua execution, Go can inspect the same
  `*Enemy` pointer and observe all mutations made through the adapter. The
  Go object and the Lua table are views of the same underlying state.

---

## Extension Guidance

> Extend behavior in Lua, not in Go.

To add new AI behaviors, combat modifiers, or status effects:

1. Add methods to the `Enemy` class in `enemy.lua`
2. Compose multiple enemies using Lua tables and loops
3. If new state queries are needed, add a Go method to `enemy.go` and expose it
   through `lua_enemy.go`

Do not embed Lua logic in Go. Do not add Lua-specific fields to the Go struct.
The adapter is the only place where Go and Lua meet.
