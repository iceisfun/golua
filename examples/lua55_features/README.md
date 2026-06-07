# Lua 5.5 Language Features

Demonstrates Lua 5.5 features beyond global declarations: named vararg
parameters, prefix attribute syntax, `table.create`, read-only for-loop
control variables, and `error(nil)` behavior.

## Run

```bash
go run ./examples/lua55_features
```

---

## Named Vararg Parameters

In Lua 5.4, collecting variadic arguments required `{...}` or `select`. Lua 5.5
adds named vararg parameters that automatically pack arguments into a table with
an `n` field:

```lua
function show(... args)
    print(args.n)     -- number of arguments (including nils)
    print(args[1])    -- first argument
    print(args[2])    -- second argument
end

show(10, 20, 30)      -- args = {10, 20, 30, n=3}
show()                -- args = {n=0}
```

### With regular parameters

Fixed parameters come first, then `... name` collects the rest:

```lua
function log(level, ... messages)
    local parts = {}
    for i = 1, messages.n do
        parts[i] = tostring(messages[i])
    end
    print("[" .. level .. "] " .. table.concat(parts, " "))
end

log("INFO", "Server started on port", 8080)
-- [INFO] Server started on port 8080
```

### Preserves nil arguments

The `n` field counts all arguments including nils, unlike the `#` operator which
stops at the first nil:

```lua
function count(... args)
    print(args.n)     -- 4
    print(#args)      -- 1 (stops at first nil)
end

count(1, nil, 3, nil)
```

### Coexists with `...`

The `...` expression still works alongside the named table:

```lua
function forward(... args)
    print(...)             -- expands all arguments
    print(args[1], args[2]) -- indexed access
end
```

### Named vararg is const

The named parameter itself cannot be reassigned (the table contents can be
modified):

```lua
function f(... t)
    t = {}            -- COMPILE ERROR: attempt to assign to const variable 't'
    t[1] = "ok"       -- fine: modifying the table is allowed
end
```

---

## Prefix Attribute Syntax

Lua 5.4 placed attributes after the variable name: `local x <const> = 1`.
Lua 5.5 also supports the attribute before the name, attached to `local`:

```lua
-- Prefix syntax (new in 5.5)
local<const> MAX = 100
local<const> GREETING = "Hello"
local<close> f = io.open("file.txt")

-- Postfix syntax (still works)
local LIMIT <const> = 50
local g <close> = io.open("other.txt")
```

When using prefix syntax with multiple names, the attribute applies to all:

```lua
local<const> A, B, C = 1, 2, 3
-- All three are const
```

Assigning to a const variable is a compile-time error:

```lua
local<const> X = 1
X = 2                 -- COMPILE ERROR: attempt to assign to const variable 'X'
```

---

## `table.create(narr, nrec)`

Preallocates a table with capacity hints. Returns an empty table with space for
`narr` array elements and `nrec` hash entries:

```lua
-- Preallocate for 1000 array elements and 2 named fields
local t = table.create(1000, 2)

-- Fill without triggering rehashes
for i = 1, 1000 do
    t[i] = i * 10
end
t.name = "example"
t.active = true

-- narr is required; nrec defaults to 0
local arrOnly = table.create(3)
```

This is useful in performance-sensitive code where the final table size is known
ahead of time.

---

## Read-Only For-Loop Control Variables

In Lua 5.5, the control variable of a `for` loop (the first variable in both
numeric and generic forms) cannot be assigned to. This is a compile-time error:

```lua
-- Numeric for: i is read-only
for i = 1, 10 do
    i = 5             -- COMPILE ERROR: attempt to assign to const variable 'i'
end

-- Generic for: k is read-only (the first variable)
for k, v in pairs(t) do
    k = "modified"    -- COMPILE ERROR: attempt to assign to const variable 'k'
end
```

### What IS allowed

Non-control variables in generic for loops can be assigned:

```lua
for k, v in pairs(t) do
    v = v * 10        -- OK: v is not the control variable
end
```

Shadowing with `local` is always fine:

```lua
for i = 1, 3 do
    local i = i + 10  -- OK: this is a new local, not the control variable
    print(i)          -- 11, 12, 13
end
```

---

## `error(nil)` Produces `"<no error object>"`

In Lua 5.4, `error(nil)` propagated `nil` as the error value, which made it
indistinguishable from "no error" in some contexts. Lua 5.5 replaces nil error
values with the string `"<no error object>"`:

```lua
local ok, msg = pcall(error, nil)
print(msg)            -- "<no error object>"
print(type(msg))      -- "string"

-- error() with no arguments behaves the same
local ok2, msg2 = pcall(error)
print(msg2)           -- "<no error object>"

-- error(false) is NOT replaced (only nil is special)
local ok3, msg3 = pcall(error, false)
print(msg3)           -- false
print(type(msg3))     -- "boolean"
```

In `xpcall`, if the error handler itself returns nil, the result is also
replaced:

```lua
local ok, msg = xpcall(
    function() error(nil) end,
    function(e) return nil end   -- handler returns nil
)
print(msg)            -- "<no error object>"
```
