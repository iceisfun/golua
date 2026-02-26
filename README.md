<p align="center">
  <img src="docs/contrib/logo2.png" alt="GoLua" width="640">
</p>

# GoLua

A Lua 5.4 interpreter written in Go, with experimental 5.5 features. Pure Go, zero dependencies, no cgo.

## Features

- Lua 5.4 language support (with experimental 5.5 features)
- Coroutines with yield/resume
- Metatables (`__index`, `__newindex`, `__call`, `__close`, etc.)
- Pattern matching (`string.match`, `string.gsub`, `string.find`)
- Go-style glob matching (`glob.match`, `glob.match_words`, `glob.match_named`)
- `<const>` and `<close>` variable attributes
- Bitwise operators (`&`, `|`, `~`, `<<`, `>>`)
- Integer division (`//`)
- Deterministic `math.random` per VM instance (seeding one VM does not affect others)
- Go interop (call Lua from Go, expose Go functions to Lua)
- Sandboxed code loading via `LuaCodeProvider`
- Sandboxed IO via `LuaIoProvider` (includes `JailedIoProvider` for read-only, directory-confined access)
- Sandboxed OS via `LuaOsProvider` (includes `DefaultOsProvider` with optional env filtering)
- Capability-gated channels for Go↔Lua message passing via `LuaChanProvider`
- Millisecond-precision timing via `LuaTimeProvider` (`time.now`, `time.since`, `time.tick`)
- Context cancellation and execution limits (call depth, stack, instructions)
- No cgo, no C dependencies, no shared object (.so/.dll) loading
- Single static binary when compiled

## Installation

```bash
go get github.com/iceisfun/golua
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/iceisfun/golua/compiler"
    "github.com/iceisfun/golua/parser"
    "github.com/iceisfun/golua/stdlib"
    "github.com/iceisfun/golua/vm"
)

func main() {
    // Parse Lua source
    source := `return 1 + 2`
    block, _ := parser.Parse("example", source)

    // Compile to bytecode
    proto, _ := compiler.Compile("example", block)

    // Create VM and load standard library
    v := vm.New()
    stdlib.Open(v)

    // Run and get results
    results, _ := v.Run(proto)
    fmt.Println(results[0].AsInt()) // Output: 3
}
```

## Examples

See the `examples/` directory for complete examples:

- **[basic](examples/basic/)** - Simple Lua execution
- **[call_lua](examples/call_lua/)** - Calling Lua functions from Go
- **[expose_go](examples/expose_go/)** - Exposing Go functions to Lua
- **[code_provider](examples/code_provider/)** - Sandboxed file loading with LuaCodeProvider
- **[jailed_io](examples/jailed_io/)** - Sandboxed IO and OS with JailedIoProvider and DefaultOsProvider
- **[debug](examples/debug/)** - Diagnostic debug with DefaultDebugProvider (not the standard Lua debug library)
- **[table](examples/table/)** - LuaTable interface and deterministic iteration
- **[chan](examples/chan/)** - Go↔Lua channels with chan.select ([go_to_lua](examples/chan/go_to_lua/), [lua_to_go](examples/chan/lua_to_go/), [multi_go_to_lua](examples/chan/multi_go_to_lua/))
- **[glob](examples/glob/)** - Go-style case-insensitive pattern matching from Go and Lua
- **[time](examples/time/)** - Millisecond timing: now, since, and periodic tick

## Go Interop

### Exposing Go Functions to Lua

```go
v := vm.New()
stdlib.Open(v)

// Register a Go function
v.SetGlobal("add", vm.NewNativeFunc(func(v *vm.VM) int {
    a := v.Get(1).AsInt()
    b := v.Get(2).AsInt()
    v.Set(0, vm.NewInt(a + b))
    return 1 // number of return values
}))
```

### Calling Lua Functions from Go

```go
// Get a Lua function
fn := v.GetGlobal("myFunction")

// Call it with arguments
results, err := v.ProtectedCall(fn, []vm.Value{
    vm.NewInt(10),
    vm.NewString("hello"),
})
```

### Sandboxed Code Loading

Implement `LuaCodeProvider` to control what Lua can load:

```go
type MyProvider struct{}

func (p *MyProvider) LoadChunk(name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
    // Validate and load the requested chunk
    // Return source, display name, and error
}

func (p *MyProvider) Capabilities() vm.LuaLoaderCaps {
    return vm.LuaLoaderCaps{
        AllowDofile:   true,
        AllowLoadfile: true,
    }
}

// Use it
v := vm.New()
v.SetCodeProvider(&MyProvider{})
stdlib.Open(v)
```

### Sandboxed IO and OS

```go
v := vm.New()

// Read-only file access, confined to a directory
v.SetIoProvider(vm.NewJailedIoProvider("/path/to/allowed/dir"))

// OS functions (clock, time, date, getenv)
v.SetOsProvider(vm.NewDefaultOsProvider())

// Or restrict which env vars are visible
v.SetOsProvider(vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
}))

stdlib.Open(v)
```

### Sandboxed Debug

Diagnostic-only debug functions (not the standard Lua debug library):

```go
v := vm.New()
v.SetDebugProvider(vm.NewDefaultDebugProvider())
stdlib.Open(v)
// Lua now has debug.traceback, debug.stackdepth, debug.where
// No hooks, no local/upvalue mutation, no bytecode inspection
```

### Context Cancellation and Execution Limits

Stop runaway scripts with cooperative context cancellation and execution limits:

```go
// Cancel infinite loops via context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

v := vm.New(vm.WithContext(ctx))
stdlib.Open(v)

// Or use setters
v.SetContext(ctx)

// Limit call depth, stack size, and instructions
v.SetLimits(vm.Limits{
    MaxCallDepth:    200,     // Prevent deep recursion
    MaxStackSlots:   10000,   // Bound memory growth
    MaxInstructions: 1000000, // Bound CPU (checkpoint visits)
    MaxMetaDepth:    500,     // Limit __index/__newindex chain depth (default: 2000)
    CompilerLimits: compiler.CompilerLimits{
        MaxVars:   200, // Max locals per function (default: 200)
        MaxRegs:   249, // Max registers per function (default: 249, hard cap: 249)
        MaxUpvals: 255, // Max upvalues per function (default: 255, hard cap: 255)
    },
})

// Or configure metatable depth independently
v = vm.New(vm.WithMaxMetaDepth(500))
v.SetMaxMetaDepth(500) // runtime setter equivalent
```

The VM checks for cancellation at backedges (loop iterations), function calls, and tail calls. No per-instruction overhead is added unless `MaxInstructions` is set. Context and limits are inherited by coroutine VMs. Errors from limits are catchable by `pcall`.

`MaxMetaDepth` bounds the length of `__index` and `__newindex` table-to-table chains to prevent infinite loops from metatable cycles. The default is 2000, matching Lua 5.4's `MAXTAGLOOP`. A value of 0 means "use the default". Function metamethods are not affected by this limit.

`CompilerLimits` enforces Lua 5.4 compile-time limits on locals, registers, and upvalues per function. These apply to `load()` and `dofile()` calls within the VM. Zero values use Lua 5.4 defaults. Limits can also be passed directly to `compiler.Compile()` via `compiler.WithLimits()`.

### Channels (Go↔Lua Message Passing)

Channels provide safe, capability-gated communication between Go and running Lua scripts. No goroutines or shared memory are exposed to Lua.

```go
// Create a provider and channel on the Go side
provider := vm.NewDefaultChanProvider()
events := provider.NewChannel(0) // unbuffered

// Set up the VM
v := vm.New()
v.SetChanProvider(provider)
stdlib.Open(v)

// Pass the channel into Lua
v.SetGlobal("events", stdlib.WrapChannel(v, events))

// Go goroutine sends events
go func() {
    events.Send(nil, vm.NewString("hello from Go"))
    events.Close()
}()

v.Run(proto) // Lua reads with events:recv()
```

Lua API:

| Function | Description |
|---|---|
| `chan.make(size?)` | Create a new channel (0 = unbuffered) |
| `chan.select(ch1, ..., timeout?)` | Receive from any ready channel; returns `idx, val, ok` |
| `ch:send(val)` | Blocking send (panics on interrupt or closed channel) |
| `ch:recv()` | Blocking receive; returns `val, ok` (`ok=false` when closed and drained) |
| `ch:close()` | Close the channel (panics if already closed) |
| `ch:try_send(val)` | Non-blocking send; returns `bool` |
| `ch:try_recv()` | Non-blocking receive; returns `val, ok, received` |

`chan.select` returns a 1-based index of the channel that fired, or `0` on timeout. Blocking operations respect context cancellation and call `CheckInterrupt()` after waking.

The `chan` table is **absent by default** (`chan == nil`). It only appears when the host sets a `LuaChanProvider` before calling `stdlib.Open()`. Channels from different providers are rejected by `chan.select` (VM boundary safety). The convenience function `stdlib.ProvideChan(v)` sets up a `DefaultChanProvider` and opens the module in one call.

### Time (Non-Standard)

Millisecond-precision timing for benchmarking and periodic triggers:

```go
v := vm.New()
v.SetTimeProvider(vm.NewDefaultTimeProvider())
stdlib.Open(v)
```

```lua
local start = time.now()           -- current time in ms
-- ... work ...
print(time.since(start) .. "ms")   -- elapsed ms

-- periodic trigger: true once per interval, false otherwise
for i = 1, math.huge do
    if time.tick(1000) then print("once per second") end
end

-- explicit key (shared across callsites)
if time.tick("heartbeat", 500) then send_heartbeat() end
```

| Function | Description |
|---|---|
| `time.now()` | Current time in milliseconds (integer) |
| `time.since(t)` | Milliseconds elapsed since `t` |
| `time.tick([name,] ms)` | Returns `true` once per `ms` interval, `false` otherwise |

When `name` is omitted, `time.tick` auto-keys by callsite (`source:line`), so each call location gets an independent timer. The `time` table is **absent by default** and only appears when the host sets a `LuaTimeProvider`.

### LuaTable Interface

Tables implement the `LuaTable` interface, which is the contract used by the VM and stdlib:

```go
type LuaTable interface {
    Get(key Value) Value
    Set(key Value, val Value)
    Delete(key Value)
    Next(key Value) (nextKey Value, val Value)
    Len() int
    Metatable() LuaTable
    SetMetatable(mt LuaTable)
}
```

The default `*Table` implementation uses an ordered keys slice for the hash part, so `next()`/`pairs()` iteration is deterministic (insertion-ordered). This avoids the non-deterministic `range map` behavior in Go. Mutation during iteration is not safe; inserting or deleting keys may skip entries or produce duplicates. Tables have no implicit thread safety -- concurrent read+write requires external synchronization.

## Standard Library

`stdlib.Open(v)` registers all standard modules. Capability-gated modules only appear when their provider is set.

| Module | Requires Provider | Description |
|--------|-------------------|-------------|
| `string` | No | Pattern matching, formatting, byte manipulation |
| `math` | No | Math functions with per-VM deterministic random |
| `table` | No | Table manipulation (sort, concat, insert, remove, move, pack, unpack) |
| `coroutine` | No | Coroutine creation and control |
| `utf8` | No | UTF-8 encoding/decoding (strict mode) |
| `bit32` | No | Lua 5.2 bitwise compat library |
| `glob` | No | Case-insensitive Go-style pattern matching (`match`, `match_words`, `match_named`) |
| `io` | `LuaIoProvider` | File I/O (absent by default) |
| `os` | `LuaOsProvider` | OS functions: clock, time, date, getenv (absent by default) |
| `debug` | `LuaDebugProvider` | Diagnostic-only: traceback, stackdepth, where (absent by default) |
| `chan` | `LuaChanProvider` | Go↔Lua message passing channels (absent by default) |
| `time` | `LuaTimeProvider` | Millisecond timing: now, since, periodic tick (absent by default) |

## Security Model

GoLua is sandboxed by default. The VM starts with no access to the host system. Capabilities are granted explicitly by the host via providers.

```
┌────────────────────────────────────────────────────────────┐
│                         Host (Go)                          │
│                                                            │
│   ┌──────────┐ ┌──────────┐ ┌────────────┐ ┌────────────┐  │
│   │IoProvider│ │OsProvider│ │ChanProvider│ │TimeProvider│  │
│   │(optional)│ │(optional)│ │(optional)  │ │(optional)  │  │
│   └────┬─────┘ └────┬─────┘ └─────┬──────┘ └─────┬──────┘  │
│        │            │             │              │         │
│   ┌────▼────────────▼─────────────▼──────────────▼──────┐  │
│   │                  VM  (sandbox)                      │  │
│   │                                                     │  │
│   │  string, math, table, coroutine, glob               │  │
│   │  io*, os*, debug*, chan*, time*                     │  │
│   │                          (* = provider-gated)       │  │
│   └─────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

- No filesystem access unless explicitly provided
- No OS access unless explicitly provided
- No native code loading
- No ambient authority

The default IO provider (`JailedIoProvider`) enforces:
- root confinement
- read-only access
- path traversal prevention

## Running Tests

```bash
go test ./...           # All tests
go test ./tests/...     # Lua script tests only
go test -v ./tests/...  # Verbose output
```

## Project Structure

```
├── ast/           # Abstract syntax tree definitions
├── compiler/      # Bytecode compiler
├── glob/          # Go-style glob matching package, diverges from standard lua
├── lexer/         # Lexical analyzer
├── parser/        # Lua parser
├── stdlib/        # Standard library implementation
├── token/         # Token definitions
├── vm/            # Virtual machine
├── tests/         # Lua test files
├── examples/      # Usage examples
└── cmd/
    ├── lua/       # Lua interpreter CLI
    └── luac/      # Bytecode compiler CLI
```

## CLI Usage

```bash
# Run a Lua script
go run ./cmd/lua script.lua

# Execute inline code
go run ./cmd/lua -e "print(1 + 1)"

# Run with a 500ms execution timeout
go run ./cmd/lua --timeout 500 script.lua

# Pass arguments to a script (available as `arg[1]`, `arg[2]`, ...)
go run ./cmd/lua script.lua foo bar

# Compile and show bytecode
go run ./cmd/luac script.lua
```

## Limitations

- No loading of C shared objects (`.so`/`.dll`) - this is by design
- No `require` with C modules
- No `io.stdin`/`io.stdout`/`io.stderr` in the library by default (the CLI at `cmd/lua` provides full stdio via its environment, but `vm.New()` does not to maintain the sandbox)
- No `io.write` in `JailedIoProvider` (read-only by design)
- No standard Lua debug library (diagnostic-only `debug.traceback`, `debug.stackdepth`, `debug.where` available via `LuaDebugProvider`)

## Contributing

PRs welcome. Run `go test ./...` before submitting.

## License

MIT
