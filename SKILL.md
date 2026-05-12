---
name: golua
description: Embed Lua 5.5 in Go applications using github.com/iceisfun/golua. Covers VM setup, native function binding, table manipulation, metatables, sandboxing, and provider-based capability control.
license: MIT
compatibility: claude-code, opencode
metadata:
  language: go
  domain: lua-embedding
---

# GoLua Skill

Use this when helping someone who imported `github.com/iceisfun/golua` and wants to embed Lua in a Go application.

## SKILLS

Copy-paste block for an AI assistant:

```text
SKILLS:
- GoLua is an embeddable, sandbox-first Lua 5.5 runtime for Go.
- Most hosts only need five steps: parser.Parse -> compiler.Compile -> vm.New -> stdlib.Open -> v.Run.
- A fresh VM is sandboxed by default. `io`, `os`, `debug`, `time`, `chan`, `exec`, and `http` are not available unless the host enables them.
- Main host tasks:
  - run a Lua chunk once
  - compile once and reuse the compiled chunk
  - expose Go functions with vm.NewNativeFunc
  - expose Go-owned structs as Lua tables
  - call Lua functions from Go with v.ProtectedCall
  - pass tables between Go and Lua
  - raise Lua-visible errors with panic(&vm.LuaError{Value: ...})
- Native function rules:
  - Lua args are 1-indexed with v.Get(1), v.Get(2), ...
  - return values are 0-indexed with v.Set(0), v.Set(1), ...
  - return the number of Lua results
  - use v.ArgCount() for variadic functions
- Lua 5.5 features: `global` keyword for explicit global declarations, named varargs (`... args`), `local<const>`/`local<close>` prefix-attribute syntax, read-only for-loop variables, `table.create(narr, nrec)`, `error(nil)` returns `"<no error object>"`.
- Removed APIs (Lua 5.5): math.atan2, math.cosh, math.sinh, math.tanh, math.log10, math.pow, debug.setcstacklimit are all removed (nil).
- Prefer explicit type checks like IsString, IsNumber, IsTable before calling AsString, AsInt, AsTable.
- If Go owns mutable state, expose closures that capture the Go pointer. If Lua just needs data, return a plain table snapshot.
- Value constructors: vm.NewInt(int64), vm.NewFloat(float64), vm.NewString(string), vm.NewBool(bool), vm.NewTable(*Table), vm.NewNativeFunc(NativeFunc). Pre-built: vm.Nil, vm.True, vm.False.
- vm.ValueToString(val) converts any Value to a printable string.
- Tables support metatables: tbl.SetMetatable(mt) / tbl.Metatable(). Set __add, __tostring, __index, __newindex, __len, __eq, __lt, __le, __call, __concat etc. as table fields.
- Table.Get(key) is raw access (like rawget). Use v.TableGet(tbl, key) for __index-aware access (like tbl[key] in Lua). This matters for class instances.
- Source directives: `directives.Parse(src)` extracts `-- @key value` header metadata for embedder-defined annotations like `-- @tick 30s`. Pure source-level (no lexer/VM coupling), header-only, non-standard Lua (reference Lua sees them as ordinary comments). The host defines what keys mean — the package is policy-free.
```

## What You Usually Need To Know

If a user just added GoLua to their app, the useful mental model is small:

1. Parse Lua source.
2. Compile it to bytecode.
3. Create a VM.
4. Open the standard library.
5. Run the compiled chunk.

That is the core path most integrations start from.

## Lua 5.5 Language Features

GoLua implements Lua 5.5. Key language changes from 5.4:

- **`global` soft keyword**: Global variable declarations can use `global x = 10` for explicit intent with compile-time checking. Undeclared global access produces a compile warning/error when global declarations are present in the chunk.
- **Named vararg parameters**: `function f(... args)` packs variadic arguments into a table `args` with an `n` field holding the count, replacing the need for `{...}` and `select("#", ...)`.
- **Prefix-attribute syntax**: Local variable attributes can use `local<const> x = 10` and `local<close> f = io.open(...)` in addition to the 5.4 postfix syntax (`local x <const> = 10`). Both forms are supported.
- **Read-only for-loop control variables**: The control variable of a `for` loop (both numeric and generic) is read-only; assigning to it is a compile-time error.
- **`table.create(narr, nrec)`**: Preallocate a table with `narr` array slots and `nrec` hash slots for performance-sensitive code.
- **`error(nil)` returns `"<no error object>"`**: Passing nil to `error()` produces the string `"<no error object>"` instead of propagating nil.
- **Removed APIs**: The following functions from Lua 5.4 are removed in golua's 5.5 implementation: `math.atan2`, `math.cosh`, `math.sinh`, `math.tanh`, `math.log10`, `math.pow`, `debug.setcstacklimit`. Use `math.atan` instead of `math.atan2`.

## Smallest Useful Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/iceisfun/golua/v2/compiler"
    "github.com/iceisfun/golua/v2/parser"
    "github.com/iceisfun/golua/v2/stdlib"
    "github.com/iceisfun/golua/v2/vm"
)

func main() {
    source := `return 1 + 2`

    block, err := parser.Parse("example", source)
    if err != nil {
        log.Fatal(err)
    }

    proto, err := compiler.Compile("example", block)
    if err != nil {
        log.Fatal(err)
    }

    v := vm.New()
    stdlib.Open(v)

    results, err := v.Run(proto)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(results[0].AsInt())
}
```

## Value Constructors

| Constructor | Go type | Lua type |
|------------|---------|----------|
| `vm.NewInt(i int64)` | `int64` | integer |
| `vm.NewFloat(f float64)` | `float64` | float |
| `vm.NewString(s string)` | `string` | string |
| `vm.NewBool(b bool)` | `bool` | boolean |
| `vm.NewTable(t *vm.Table)` | `*vm.Table` | table |
| `vm.NewNativeFunc(f NativeFunc)` | `func(*vm.VM) int` | function |
| `vm.NewFunction(c *Closure)` | `*vm.Closure` | function |
| `vm.Nil` | — | nil |
| `vm.True` / `vm.False` | — | boolean |

Use `vm.ValueToString(val)` to convert any Value to a printable string (useful in REPLs, logging, debugging).

## One-Shot Run Vs Reuse

Use a one-shot run when the script is transient and you do not care about reusing compiled code or VM state.

Reuse the compiled chunk when you want to run the same script many times:

```go
block, err := parser.Parse("worker", source)
if err != nil {
    panic(err)
}

proto, err := compiler.Compile("worker", block)
if err != nil {
    panic(err)
}

for _, name := range []string{"a", "b", "c"} {
    v := vm.New()
    stdlib.Open(v)
    v.SetGlobal("name", vm.NewString(name))
    if _, err := v.Run(proto); err != nil {
        panic(err)
    }
}
```

Reuse the same VM when you want globals, loaded modules, or Lua-defined functions to persist:

```go
v := vm.New()
stdlib.Open(v)

setup := `
    total = 0
    function add(n)
        total = total + n
        return total
    end
`

block, err := parser.Parse("setup", setup)
if err != nil {
    panic(err)
}

proto, err := compiler.Compile("setup", block)
if err != nil {
    panic(err)
}

if _, err := v.Run(proto); err != nil {
    panic(err)
}

fn := v.GetGlobal("add")
results, err := v.ProtectedCall(fn, []vm.Value{vm.NewInt(5)})
if err != nil {
    panic(err)
}

fmt.Println(results[0].AsInt())
```

Rule of thumb:

- fresh VM for isolation
- reused compiled chunk for speed
- reused VM for persistent Lua state

## Expose A Go Function To Lua

```go
v.SetGlobal("repeat_string", vm.NewNativeFunc(func(v *vm.VM) int {
    s := v.Get(1)
    n := v.Get(2)

    if !s.IsString() {
        panic(&vm.LuaError{Value: vm.NewString("bad argument #1: expected string")})
    }
    if !n.IsNumber() {
        panic(&vm.LuaError{Value: vm.NewString("bad argument #2: expected number")})
    }

    out := strings.Repeat(s.AsString(), int(n.AsInt()))
    v.Set(0, vm.NewString(out))
    return 1
}))
```

Native functions should validate their own inputs. That makes the Lua-facing API much easier for an assistant to reason about.

## Accept Different Lua Argument Types

Do not assume the type. Check first.

```go
v.SetGlobal("describe", vm.NewNativeFunc(func(v *vm.VM) int {
    arg := v.Get(1)

    switch {
    case arg.IsNil():
        v.Set(0, vm.NewString("nil"))
    case arg.IsBool():
        v.Set(0, vm.NewString(fmt.Sprintf("bool:%v", arg.AsBool())))
    case arg.IsNumber():
        v.Set(0, vm.NewString(fmt.Sprintf("number:%v", arg.AsFloat())))
    case arg.IsString():
        v.Set(0, vm.NewString("string:"+arg.AsString()))
    case arg.IsTable():
        v.Set(0, vm.NewString("table"))
    case arg.IsCallable():
        v.Set(0, vm.NewString("function"))
    default:
        v.Set(0, vm.NewString(arg.Type()))
    }

    return 1
}))
```

For variadic input:

```go
v.SetGlobal("sum_all", vm.NewNativeFunc(func(v *vm.VM) int {
    var sum int64
    for i := 1; i <= v.ArgCount(); i++ {
        arg := v.Get(i)
        if !arg.IsNumber() {
            panic(&vm.LuaError{Value: vm.NewString(fmt.Sprintf("bad argument #%d: expected number, got %s", i, arg.Type()))})
        }
        sum += arg.AsInt()
    }
    v.Set(0, vm.NewInt(sum))
    return 1
}))
```

## Expose A Struct As A Table Of Methods

Use this when Go owns the live state.

```go
type Counter struct {
    name  string
    value int64
}

func CounterToLua(c *Counter) *vm.Table {
    t := vm.NewEmptyTable()

    t.SetString("get_name", vm.NewNativeFunc(func(v *vm.VM) int {
        v.Set(0, vm.NewString(c.name))
        return 1
    }))

    t.SetString("get_value", vm.NewNativeFunc(func(v *vm.VM) int {
        v.Set(0, vm.NewInt(c.value))
        return 1
    }))

    t.SetString("add", vm.NewNativeFunc(func(v *vm.VM) int {
        delta := v.Get(1)
        if !delta.IsNumber() {
            panic(&vm.LuaError{Value: vm.NewString("bad argument #1: expected number")})
        }
        c.value += delta.AsInt()
        v.Set(0, vm.NewInt(c.value))
        return 1
    }))

    return t
}

counter := &Counter{name: "hits"}
v.SetGlobal("counter", vm.NewTable(CounterToLua(counter)))
```

Lua usage:

```lua
print(counter.get_name())
print(counter.add(3))
```

This pattern is great when Go should remain the source of truth.

## Expose A Struct As Plain Table Values

Use this when Lua mainly needs data, not live Go-backed behavior.

```go
type Thing struct {
    Name string
    Age  int
}

func ThingToLuaSnapshot(tg Thing) *vm.Table {
    t := vm.NewEmptyTable()
    t.SetString("name", vm.NewString(tg.Name))
    t.SetString("age", vm.NewInt(int64(tg.Age)))
    return t
}

v.SetGlobal("thing", vm.NewTable(ThingToLuaSnapshot(Thing{
    Name: "gizmo",
    Age:  4,
})))
```

Lua usage:

```lua
print(thing.name)
print(thing.age)
```

This is simpler for consumers, but it is a snapshot unless you keep it synchronized yourself.

## Mix Values And Methods

This often gives the nicest Lua API.

```go
func ThingToLua(tg *Thing) *vm.Table {
    t := vm.NewEmptyTable()
    t.SetString("name", vm.NewString(tg.Name))
    t.SetString("age", vm.NewInt(int64(tg.Age)))

    t.SetString("rename", vm.NewNativeFunc(func(v *vm.VM) int {
        name := v.Get(1)
        if !name.IsString() {
            panic(&vm.LuaError{Value: vm.NewString("bad argument #1: expected string")})
        }
        tg.Name = name.AsString()
        t.SetString("name", vm.NewString(tg.Name))
        return 0
    }))

    return t
}
```

Now Lua gets both `thing.name` and `thing.rename("new name")`.

## Raw Vs Metamethod-Aware Table Access

`Table.Get()` is **raw access** — it does not walk the `__index` chain. This is
equivalent to Lua's `rawget()` and is the correct behavior for direct table
operations.

When you need Lua-style indexing that respects `__index` (table or function),
use the VM methods instead:

```go
// Raw access (no metamethods) — like rawget()
val := tbl.Get(vm.NewString("key"))

// Metamethod-aware access — like tbl["key"] in Lua
val, err := v.TableGet(tbl, vm.NewString("key"))

// Metamethod-aware integer access — like tbl[1] in Lua
val, err := v.TableGetInt(tbl, 1)

// Metamethod-aware write — like tbl["key"] = val in Lua
err := v.SetIndexValue(vm.NewTable(tbl), vm.NewString("key"), val)
```

This matters when working with Lua OOP patterns. Instances created via
`setmetatable({}, Class)` store methods on the class, not the instance.
`Table.Get()` on the instance will return nil for inherited methods:

```go
results, _ := v.Run(proto) // Lua returns an instance with methods via __index
instance := results[0].AsTable()

// WRONG: raw access, misses inherited methods
method := instance.Get(vm.NewString("greet")) // nil!

// RIGHT: walks __index chain
method, err := v.TableGet(instance, vm.NewString("greet")) // found
```

Rule of thumb:
- Use `tbl.Get()` when you know the key is on the table itself (config tables, plain data)
- Use `v.TableGet()` when the table might use metatables (class instances, proxies)

## Accept A Table Passed From Lua

Check `IsTable()` first, then read fields from the `LuaTable`.

```go
v.SetGlobal("configure", vm.NewNativeFunc(func(v *vm.VM) int {
    arg := v.Get(1)
    if !arg.IsTable() {
        panic(&vm.LuaError{Value: vm.NewString("bad argument #1: expected table")})
    }

    tbl := arg.AsTable()
    name := tbl.Get(vm.NewString("name"))
    enabled := tbl.Get(vm.NewString("enabled"))
    retries := tbl.Get(vm.NewString("retries"))

    if !name.IsString() {
        panic(&vm.LuaError{Value: vm.NewString("config.name must be a string")})
    }
    if !enabled.IsBool() {
        panic(&vm.LuaError{Value: vm.NewString("config.enabled must be a boolean")})
    }
    if !retries.IsNumber() {
        panic(&vm.LuaError{Value: vm.NewString("config.retries must be a number")})
    }

    fmt.Println(name.AsString(), enabled.AsBool(), retries.AsInt())
    return 0
}))
```

Lua:

```lua
configure({
    name = "worker-a",
    enabled = true,
    retries = 3,
})
```

If you need to iterate the table:

```go
tbl := v.Get(1).AsTable()
var key vm.Value = vm.Nil
for {
    nextKey, value, err := tbl.Next(key)
    if err != nil {
        panic(&vm.LuaError{Value: vm.NewString(err.Error())})
    }
    if nextKey.IsNil() {
        break
    }
    fmt.Println(vm.ValueToString(nextKey), vm.ValueToString(value))
    key = nextKey
}
```

`vm.ValueToString` is the standard way to get a human-readable representation of any Lua value from Go. It handles nil, bool, int, float, and string directly; other types produce a type-and-pointer format like Lua's `tostring()`.

## Return A Table To Lua

```go
v.SetGlobal("make_point", vm.NewNativeFunc(func(v *vm.VM) int {
    t := vm.NewEmptyTable()
    t.SetString("x", v.Get(1))
    t.SetString("y", v.Get(2))
    v.Set(0, vm.NewTable(t))
    return 1
}))
```

## Call Lua From Go

Use `ProtectedCall` when Go wants to invoke a Lua callback.

```go
fn := v.GetGlobal("handler")
if !fn.IsCallable() {
    panic("handler is not callable")
}

results, err := v.ProtectedCall(fn, []vm.Value{
    vm.NewString("hello"),
    vm.NewInt(42),
})
if err != nil {
    panic(err)
}

fmt.Println(results[0])
```

## Error Back To Lua From Go

Inside a native function, return Lua-visible errors by panicking with `*vm.LuaError`.

Simple string error:

```go
panic(&vm.LuaError{Value: vm.NewString("bad argument #1: expected table")})
```

Structured Lua error object:

```go
errTbl := vm.NewEmptyTable()
errTbl.SetString("code", vm.NewString("E_BAD_INPUT"))
errTbl.SetString("message", vm.NewString("invalid payload"))
panic(&vm.LuaError{Value: vm.NewTable(errTbl)})
```

Lua can catch either with `pcall`.

```lua
local ok, err = pcall(failing_call)
if not ok then
    print("caught:", err)
end
```

Use this for user-facing validation failures. Do not use plain Go panics for normal Lua argument errors.

## Metatables

GoLua supports full Lua 5.5 metatables. Set a metatable on any table to define
operator overloads, custom indexing, and string conversion.

### Basic metatable with __tostring and __add

```go
func NewVec2(x, y float64) *vm.Table {
    t := vm.NewEmptyTable()
    t.SetString("x", vm.NewFloat(x))
    t.SetString("y", vm.NewFloat(y))
    return t
}

func Vec2Meta() *vm.Table {
    mt := vm.NewEmptyTable()

    mt.SetString("__tostring", vm.NewNativeFunc(func(v *vm.VM) int {
        self := v.Get(1).AsTable()
        x := self.Get(vm.NewString("x")).AsFloat()
        y := self.Get(vm.NewString("y")).AsFloat()
        v.Set(0, vm.NewString(fmt.Sprintf("vec2(%g, %g)", x, y)))
        return 1
    }))

    mt.SetString("__add", vm.NewNativeFunc(func(v *vm.VM) int {
        a := v.Get(1).AsTable()
        b := v.Get(2).AsTable()
        ax := a.Get(vm.NewString("x")).AsFloat()
        ay := a.Get(vm.NewString("y")).AsFloat()
        bx := b.Get(vm.NewString("x")).AsFloat()
        by := b.Get(vm.NewString("y")).AsFloat()
        result := NewVec2(ax+bx, ay+by)
        result.SetMetatable(a.Metatable()) // propagate metatable
        v.Set(0, vm.NewTable(result))
        return 1
    }))

    return mt
}
```

Wire it up:

```go
meta := Vec2Meta()
a := NewVec2(1, 2)
a.SetMetatable(meta)
b := NewVec2(3, 4)
b.SetMetatable(meta)

v.SetGlobal("a", vm.NewTable(a))
v.SetGlobal("b", vm.NewTable(b))
```

Lua usage:

```lua
print(a)         --> vec2(1, 2)
local c = a + b
print(c)         --> vec2(4, 6)
```

### Supported metamethods

All standard Lua 5.5 metamethods work: `__add`, `__sub`, `__mul`, `__div`,
`__mod`, `__pow`, `__unm`, `__idiv`, `__band`, `__bor`, `__bxor`, `__bnot`,
`__shl`, `__shr`, `__eq`, `__lt`, `__le`, `__concat`, `__len`, `__index`,
`__newindex`, `__call`, `__tostring`, `__close`.

### __index for default fields or method dispatch

```go
mt.SetString("__index", vm.NewNativeFunc(func(v *vm.VM) int {
    key := v.Get(2).AsString()
    switch key {
    case "length":
        self := v.Get(1).AsTable()
        x := self.Get(vm.NewString("x")).AsFloat()
        y := self.Get(vm.NewString("y")).AsFloat()
        v.Set(0, vm.NewFloat(math.Sqrt(x*x + y*y)))
    default:
        v.Set(0, vm.Nil)
    }
    return 1
}))
```

You can also set `__index` to a table for prototype-style inheritance:

```go
methods := vm.NewEmptyTable()
methods.SetString("length", vm.NewNativeFunc(func(v *vm.VM) int {
    self := v.Get(1).AsTable()
    x := self.Get(vm.NewString("x")).AsFloat()
    y := self.Get(vm.NewString("y")).AsFloat()
    v.Set(0, vm.NewFloat(math.Sqrt(x*x + y*y)))
    return 1
}))
mt.SetString("__index", vm.NewTable(methods))
```

Lua:

```lua
print(a:length())  --> 2.2360679774998
```

Note: `a:length()` is sugar for `a.length(a)` — the colon passes the table as the first arg, so `v.Get(1)` is `self` and real arguments start at `v.Get(2)`. Dot-calls like `a.length()` do not pass `self`.

### Setting metatables from Go

```go
tbl.SetMetatable(mt)         // set metatable on a table instance
tbl.Metatable()              // get current metatable (nil if none)
v.SetStringMeta(mt)          // set type metatable for all strings
v.GetMetafield(val, "__len") // look up a specific metamethod
```

## Output Capture

Capture `print()` output in memory:

```go
v := vm.New(vm.WithCaptureOutput(true))
stdlib.Open(v)

_, _ = v.Run(proto)
fmt.Println(v.OutputLines())
fmt.Println(v.LastOutput())
v.ClearOutput()
```

Or route output through your own logger:

```go
type Logger struct{}

func (l *Logger) Print(ctx context.Context, msg string) { log.Printf("lua: %s", msg) }
func (l *Logger) Warn(ctx context.Context, msg string)  { log.Printf("lua warn: %s", msg) }

v := vm.New()
v.SetPrintProvider(&Logger{})
stdlib.Open(v)
```

## Sandbox And Optional Capabilities

A fresh `vm.New()` is intentionally limited. Modules that touch the outside world
are absent until the host explicitly enables them by setting a provider before
calling `stdlib.Open`:

| Lua module | Provider setter | Default implementation |
|------------|----------------|----------------------|
| `io`       | `v.SetIoProvider(...)` | `vm.NewFullIoProvider(root)` (full access) or `vm.NewJailedIoProvider(root)` (read-only, directory-jailed) |
| `os`       | `v.SetOsProvider(...)` | `vm.NewDefaultOsProvider()` |
| `debug`    | `v.SetDebugProvider(...)` | `vm.NewDefaultDebugProvider()` |
| `time`     | `v.SetTimeProvider(...)` | `vm.NewDefaultTimeProvider()` |
| `chan`      | `v.SetChanProvider(...)` | `vm.NewDefaultChanProvider()` |
| `exec`     | `v.SetProcessProvider(...)` | `vm.NewDefaultProcessProvider()` |
| `os.execute` | `v.SetExecProvider(...)` | `vm.NewDefaultExecProvider()` (requires OS provider too) |
| `os.exit`  | `v.SetExitHandler(...)` | host-defined |
| `loadfile`/`dofile` | `v.SetCodeProvider(...)` | host-defined (see below) |
| `package.loadlib` | `v.SetLoadLibProvider(...)` | host-defined |
| print/warn routing | `v.SetPrintProvider(...)` | built-in (writes to stdout) |

The HTTP module lives in a separate package and must be opened explicitly:

```go
import (
    "github.com/iceisfun/golua/v2/stdlib"
    gohttp "github.com/iceisfun/golua/v2/stdlib/http"
)

v := vm.New()
stdlib.Open(v)
gohttp.Open(v)
```

Typical setup enabling common capabilities:

```go
v := vm.New()
v.SetIoProvider(vm.NewFullIoProvider("/app/data"))
v.SetOsProvider(vm.NewDefaultOsProvider())
v.SetDebugProvider(vm.NewDefaultDebugProvider())
stdlib.Open(v)
```

Typical setup for bounded execution:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

v := vm.New(
    vm.WithContext(ctx),
    vm.WithLimits(vm.Limits{
        MaxCallDepth:    200,
        MaxStackSlots:   10000,
        MaxInstructions: 1000000,
    }),
)
stdlib.Open(v)
```

`vm.New()` defaults to `context.Background()`, so `v.Context()` is never nil.
All provider interface methods receive this context, allowing them to respect
cancellation, deadlines, and request-scoped values.

## VM Lifecycle: Close, Initializable, Shutdownable

Providers can optionally implement lifecycle interfaces for setup and teardown:

```go
// Initializable is called when a provider is set on a VM.
type Initializable interface {
    Initialize(ctx context.Context) error
}

// Shutdownable is called when VM.Close is invoked.
type Shutdownable interface {
    Shutdown(ctx context.Context) error
}
```

Call `v.Close(ctx)` when you are done with a VM to let providers release
resources (close connections, stop goroutines, flush buffers, etc.):

```go
v := vm.New(vm.WithContext(ctx))
v.SetIoProvider(vm.NewFullIoProvider("/app/data"))
stdlib.Open(v)

_, err := v.Run(proto)
// ... handle err ...

if err := v.Close(ctx); err != nil {
    log.Printf("cleanup error: %v", err)
}
```

`Close` iterates all set providers and calls `Shutdown` on any that implement
`Shutdownable`. It returns the first error encountered. Providers that do not
implement `Shutdownable` are silently skipped.

A custom provider using both interfaces:

```go
type DBProvider struct {
    db *sql.DB
}

func (p *DBProvider) Initialize(ctx context.Context) error {
    return p.db.PingContext(ctx)
}

func (p *DBProvider) Shutdown(ctx context.Context) error {
    return p.db.Close()
}
```

## Custom File Systems And Code Loading

You can implement your own providers to virtualize I/O and code loading.

### Custom IO provider

Implement `vm.LuaIoProvider` and `vm.LuaFile` to back `io.*` with any storage
(embed.FS, database blobs, in-memory buffers, network mounts, etc.).
`JailedIoProvider` is a good reference: it wraps `fs.FS` for read-only access.

```go
type EmbedIoProvider struct { fsys fs.FS }

func (p *EmbedIoProvider) Open(ctx context.Context, name, mode string) (vm.LuaFile, error) { /* ... */ }
func (p *EmbedIoProvider) Capabilities(ctx context.Context) vm.LuaIoCaps {
    return vm.LuaIoCaps{AllowRead: true, AllowWrite: false}
}
func (p *EmbedIoProvider) Stdin(ctx context.Context) vm.LuaFile  { return nil }
func (p *EmbedIoProvider) Stdout(ctx context.Context) vm.LuaFile { return nil }
func (p *EmbedIoProvider) Stderr(ctx context.Context) vm.LuaFile { return nil }
func (p *EmbedIoProvider) TmpName(ctx context.Context) (string, error) { return "", fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) TmpFile(ctx context.Context) (vm.LuaFile, error) { return nil, fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) Remove(ctx context.Context, _ string) error { return fmt.Errorf("unsupported") }
func (p *EmbedIoProvider) Rename(ctx context.Context, _, _ string) error { return fmt.Errorf("unsupported") }
```

All provider interface methods take `ctx context.Context` as their first
parameter. The VM passes its own context (from `vm.WithContext(ctx)` or
`context.Background()` by default) through to every provider call. This lets
providers respect cancellation, deadlines, and request-scoped values.

### Custom code provider

Implement `vm.LuaCodeProvider` to control what `loadfile()` and `dofile()` can
load. This is how you build a virtual module system without real filesystem
access.

```go
type InMemoryLoader struct {
    files map[string]string
}

func (l *InMemoryLoader) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
    src, ok := l.files[name]
    if !ok {
        return nil, "", fmt.Errorf("module %q not found", name)
    }
    return []byte(src), "@" + name, nil
}

func (l *InMemoryLoader) Capabilities(ctx context.Context) vm.LuaLoaderCaps {
    return vm.LuaLoaderCaps{AllowDofile: true, AllowLoadfile: true}
}
```

The caller context gives you the requesting script name, VM ID, and call depth
for audit logging or policy decisions.

## Provider Interface Reference

Each provider is set on the VM before calling `stdlib.Open`. All methods take `ctx context.Context` as their first parameter.

### LuaCodeProvider — controls dofile, loadfile, require

```go
type LuaCodeProvider interface {
    LoadChunk(ctx context.Context, name string, caller *LuaCallerContext) (source []byte, chunkName string, err error)
    Capabilities(ctx context.Context) LuaLoaderCaps
}
type LuaLoaderCaps struct { AllowDofile, AllowLoadfile bool }
```

Setter: `v.SetCodeProvider(...)` | Default: `vm.NewDirCodeProvider(root, caps)`

### LuaIoProvider — controls io.* file operations

```go
type LuaIoProvider interface {
    Open(ctx context.Context, name, mode string) (LuaFile, error)
    Capabilities(ctx context.Context) LuaIoCaps
    Stdin(ctx context.Context) LuaFile
    Stdout(ctx context.Context) LuaFile
    Stderr(ctx context.Context) LuaFile
    TmpName(ctx context.Context) (string, error)
    Remove(ctx context.Context, name string) error
    Rename(ctx context.Context, oldname, newname string) error
    TmpFile(ctx context.Context) (LuaFile, error)
}
type LuaIoCaps struct { AllowRead, AllowWrite bool }
```

Setter: `v.SetIoProvider(...)` | Defaults: `vm.NewJailedIoProvider(root)` (read-only), `vm.NewFullIoProvider(root)` (read-write)

### LuaOsProvider — controls os.clock, os.time, os.date, os.getenv, os.setlocale

```go
type LuaOsProvider interface {
    Clock(ctx context.Context) float64
    Time(ctx context.Context, dateTable *LuaTimeInput) (int64, *LuaDateTime, error)
    Date(ctx context.Context, format string, timestamp int64) (string, error)
    DateTable(ctx context.Context, timestamp int64, utc bool) *LuaDateTime
    Getenv(ctx context.Context, name string) (string, bool)
    SetLocale(ctx context.Context, locale, category string) (string, bool)
    Capabilities(ctx context.Context) LuaOsCaps
}
type LuaOsCaps struct {
    AllowTime, AllowDate, AllowGetenv, AllowTmpName bool
    AllowRemove, AllowExecute, AllowExit, AllowRename bool
}
```

Setter: `v.SetOsProvider(...)` | Defaults: `vm.NewDefaultOsProvider()`, `vm.NewFilteredOsProvider(filter)`

### LuaExecProvider — controls os.execute

```go
type LuaExecProvider interface {
    Execute(ctx context.Context, command string) (ok bool, exitType string, exitCode int)
}
```

Setter: `v.SetExecProvider(...)` | Default: `vm.NewDefaultExecProvider()`

### LuaExitHandler — controls os.exit

```go
type LuaExitHandler interface {
    Exit(ctx context.Context, code int, close bool)
}
```

Setter: `v.SetExitHandler(...)` | Default: `vm.NewDefaultExitHandler()` (panics with `*LuaExitError`)

### LuaDebugProvider — gates debug.* functions

```go
type LuaDebugProvider interface {
    Capabilities(ctx context.Context) LuaDebugCaps
}
type LuaDebugCaps struct {
    AllowTraceback, AllowStackDepth, AllowWhere, AllowGetInfo bool
    AllowGetUpvalue, AllowSetUpvalue, AllowUpvalueID bool
    AllowGetLocal, AllowSetLocal bool
    AllowGetRegistry, AllowGetMetatable, AllowSetMetatable bool
    AllowSetHook, AllowGetHook, AllowUpvalueJoin bool
    AllowSetCStackLimit, AllowGetUserValue, AllowSetUserValue bool
}
```

Setter: `v.SetDebugProvider(...)` | Default: `vm.NewDefaultDebugProvider()` (all enabled)

### LuaChanProvider — controls chan.* Go↔Lua channels

```go
type LuaChanProvider interface {
    NewChannel(ctx context.Context, size int) *LuaChannel
    Capabilities(ctx context.Context) LuaChanCaps
}
type LuaChanCaps struct {
    AllowSend, AllowRecv, AllowClose bool
    AllowSelect, AllowTrySend, AllowTryRecv bool
}
```

Setter: `v.SetChanProvider(...)` | Default: `vm.NewDefaultChanProvider()`

### LuaTimeProvider — controls time.* millisecond timing

```go
type LuaTimeProvider interface {
    Now(ctx context.Context) int64
    Tick(ctx context.Context, key string, ms int64) bool
    Once(ctx context.Context, key string) bool
}
```

Setter: `v.SetTimeProvider(...)` | Default: `vm.NewDefaultTimeProvider()`

### LuaPrintProvider — routes print()/warn() output

```go
type LuaPrintProvider interface {
    Print(ctx context.Context, msg string)
    Warn(ctx context.Context, msg string)
}
```

Setter: `v.SetPrintProvider(...)` | Default: `vm.NewDefaultPrintProvider()` (stdout/stderr)

### LuaProcessProvider — controls exec.* process spawning

```go
type LuaProcessProvider interface {
    Spawn(ctx context.Context, cmd string, args []string, opts ProcessOptions) (LuaProcess, error)
}
type ProcessOptions struct {
    Env map[string]string; Dir string
    Stdin, Stdout, Stderr, MergeStderr bool
}
```

Setter: `v.SetProcessProvider(...)` | Default: `vm.NewDefaultProcessProvider()`

### LuaLoadLibProvider — controls package.loadlib

```go
type LuaLoadLibProvider interface {
    LoadLib(ctx context.Context, path, init string, caller *LuaCallerContext) (loader NativeFunc, errmsg string, where string)
}
```

Setter: `v.SetLoadLibProvider(...)` | Default: none (returns "absent")

## Source Directives (Header Metadata)

A common embedder pattern is annotating Lua scripts with host-meaningful metadata in their header — scheduler intervals, scope names, enable/disable flags, registration hints. The `directives` sub-package parses these without involving the lexer, parser, compiler, or VM:

```go
import "github.com/iceisfun/golua/v2/directives"

f, _ := directives.Parse(source)        // never errors in v1; *File is always non-nil
if f.Has("disabled") { return }         // flag directives: ("", true)
tick, _ := f.Get("tick")                // last-wins for repeated keys
imports := f.Lookup("import")           // every value, in source order
for k, v := range f.All() { ... }       // range-over-func iterator
```

```lua
-- @tick 30s
-- @scope alias_expander
-- @disabled
-- @import shared/util
-- @import shared/log

local function run() return 42 end
```

Key rules:

- **Header only.** Parser scans the contiguous prefix of shebang, blank lines, and `--` short comments. Stops at the first code line OR the first long comment (`--[[ ]]`).
- **Non-directive comments inside the header are ignored** but do not terminate it (so `-- this is a banner` between directives is fine).
- **Repeated keys:** preserved in order. `Get` returns last; `Lookup` returns all.
- **Flag directives** (`-- @disabled`): key present with empty value. `Has` is the cleanest test.
- **Case-sensitive**, no normalization. Identifier charset: `[A-Za-z_][A-Za-z0-9_-]*`.
- **Malformed `-- @...` lines are silently ignored**, not errors. (A future `ParseStrict` may opt in to errors.)
- **No policy in the package.** `@tick`, `@scope`, `@disabled` are embedder conventions. The host parses values (e.g. `time.ParseDuration("30s")`) and decides what they mean.

When to recommend it: an embedder is hand-rolling a regex over comments to pick up `@`-tagged metadata. Replace with `directives.Parse`.

When NOT to use it: the metadata needs to bind to specific declarations (`@tick` on `function foo()`). Declaration-bound annotations are an explicit non-goal; the package is header-scoped only.

Non-standard Lua disclosure: directives are encoded in ordinary `--` comments. Reference Lua executes the same source unchanged — only the *interpretation* is golua-specific. Stripped/source-less execution carries no directive data because nothing enters the bytecode pipeline.

See `examples/directives` (minimal demo) and `examples/directive_loader` (realistic directory-scan + policy pattern).

## Guidance For AI Assistants

If you only read this file, you should still be able to help a GoLua user with normal embedding work.

Good defaults:

- reach for the five-step flow first
- use explicit native functions instead of magic/reflection
- keep Go as the source of truth for live mutable objects
- use plain table snapshots for simple data
- validate Lua inputs explicitly
- use `ProtectedCall` for Lua callbacks
- use `LuaError` for Lua-facing errors
- only bring in providers when the host actually wants those capabilities
- all provider and LuaFile interface methods take ctx context.Context as first param; vm.New() defaults to context.Background()
- call v.Close(ctx) to shut down providers that implement Shutdownable; providers may also implement Initializable

The goal of this skill is not to explain the repo internals. It is to help an assistant build correct, practical host integrations quickly.
