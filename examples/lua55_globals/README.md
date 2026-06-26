# Lua 5.5 Global Declarations

Demonstrates Lua 5.5's `global` declaration system: explicit declarations,
constant globals, wildcard mode, compile-time checking, scope inheritance,
global functions, and soft keyword behavior.

## Run

```bash
go run ./examples/lua55_globals
```

---

## How `global` Works

In Lua 5.5, every chunk starts with an implicit `global *`, meaning all global
names are allowed freely -- the same behavior as Lua 5.4. Once any explicit
`global` statement appears in a block, the implicit wildcard is voided for that
scope and only declared names are permitted.

### Declaring globals

```lua
global x, y           -- x and y are read-write globals
global print, tostring -- must declare builtins you use too

x = 10
y = 20
print(x + y)          -- 30
```

### Constant globals

```lua
global<const> PI = 3.14159
global<const> VERSION = "1.0"

print(PI)       -- OK: reading a const global
PI = 3.0        -- COMPILE ERROR: attempt to assign to const variable 'PI'
```

The `<const>` attribute can also appear after the name:

```lua
global PI <const> = 3.14159
```

### Wildcard declarations

```lua
global *              -- allow all globals, read-write (like Lua 5.4 default)
global<const> *       -- allow all globals, but read-only

x = 1                 -- OK with global *, error with global<const> *
print(x)              -- OK with either
```

### Specific overrides wildcard

When both a specific declaration and a wildcard exist, the specific one wins:

```lua
global counter        -- explicitly read-write
global<const> *       -- everything else is read-only

counter = counter + 1 -- OK: counter is explicitly rw
print(counter)        -- OK: reading is always fine
x = 1                 -- COMPILE ERROR: x falls under const wildcard
```

### Global function declarations

```lua
global function greet(name)
    return "Hello, " .. name .. "!"
end
```

This is equivalent to `global greet; greet = function(name) ... end` but
declares and assigns in one statement.

---

## What `global` Does NOT Do

### It is a soft keyword

`global` is only special at statement start. Everywhere else, it is an ordinary
name:

```lua
local global = 42     -- local variable named "global"
print(global)         -- 42

local t = { global = "field" }
print(t.global)       -- "field"

function f(global)    -- parameter named "global"
    return global * 2
end

global = 99           -- plain assignment to _ENV["global"]
print(global)         -- 99
```

### It does NOT create a new variable scope

`global` does not introduce a new binding -- it permits (or restricts) access to
keys in `_ENV`. All globals still live in the same environment table.

### It does NOT affect runtime behavior

The checking is purely at compile time. At runtime, `global x = 10` is the same
as `x = 10` (a SETTABUP into `_ENV`). The only exception is `<const>` globals
with initializers, which also emit a runtime nil-check (the value must not be
nil).

### Undeclared access is a compile-time error, not a runtime one

```lua
global x
print(y)              -- COMPILE ERROR: variable 'y' not declared
z = 1                 -- COMPILE ERROR: variable 'z' not declared
```

### Declarations propagate into nested functions

Global declarations are lexically scoped, so a nested function inherits the
enclosing chunk's declarations -- both the explicit mode and the declared
names. It does NOT start fresh with its own implicit `global *`:

```lua
global x              -- explicit mode in outer chunk

local f = function()
    x = 1             -- OK: 'x' is declared in the enclosing scope
    y = 1             -- COMPILE ERROR: 'y' not declared (explicit mode inherited)
end
```

---

## Scope Inheritance

Global declarations are block-scoped and cumulative within a block. Inner
blocks inherit outer declarations:

```lua
global x, print
x = 100

do
    global y          -- inner block adds y
    y = 200
    print(x, y)       -- both visible: 100  200
end

-- y's declaration is no longer in scope, but the outer block's
-- explicit mode is still active, so undeclared names are still errors
```

---

## Common Patterns

### Library-style module

```lua
global print, type, tostring, error, pairs, ipairs
global<const> *       -- everything else (builtins) is read-only

-- Your code here -- any typo in a global name is caught at compile time
```

### Strict mode (all globals must be declared)

```lua
global print, assert, pcall, type, error
-- Now any undeclared global is a compile error
```

### Configuration constants

```lua
global<const> MAX_RETRIES = 3
global<const> TIMEOUT = 5000
global<const> VERSION = "2.1.0"

global print, tostring
global count = 0

global function increment()
    count = count + 1
end
```
