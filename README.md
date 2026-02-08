# GoLua

A Lua 5.5 interpreter written in Go. Pure Go, zero dependencies, no cgo.

## Features

- Lua 5.5 language support
- Coroutines with yield/resume
- Metatables (`__index`, `__newindex`, `__call`, `__close`, etc.)
- Pattern matching (`string.match`, `string.gsub`, `string.find`)
- `<const>` and `<close>` variable attributes
- Bitwise operators (`&`, `|`, `~`, `<<`, `>>`)
- Integer division (`//`)
- Go interop (call Lua from Go, expose Go functions to Lua)
- Sandboxed code loading via `LuaCodeProvider`
- Sandboxed IO via `LuaIoProvider` (includes `JailedIoProvider` for read-only, directory-confined access)
- Sandboxed OS via `LuaOsProvider` (includes `DefaultOsProvider` with optional env filtering)
- Capability-gated channels for Go↔Lua message passing via `LuaChanProvider`
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
})

// Or configure metatable depth independently
v = vm.New(vm.WithMaxMetaDepth(500))
v.SetMaxMetaDepth(500) // runtime setter equivalent
```

The VM checks for cancellation at backedges (loop iterations), function calls, and tail calls. No per-instruction overhead is added unless `MaxInstructions` is set. Context and limits are inherited by coroutine VMs. Errors from limits are catchable by `pcall`.

`MaxMetaDepth` bounds the length of `__index` and `__newindex` table-to-table chains to prevent infinite loops from metatable cycles. The default is 2000, matching Lua 5.4's `MAXTAGLOOP`. A value of 0 means "use the default". Function metamethods are not affected by this limit.

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

## Security Model

golua is sandboxed by default.

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

# Compile and show bytecode
go run ./cmd/luac script.lua
```

## Limitations

- No loading of C shared objects (`.so`/`.dll`) - this is by design
- No `require` with C modules
- No `io.stdin`/`io.stdout`/`io.stderr` (no implicit stdio)
- No `io.write` in `JailedIoProvider` (read-only by design)
- No standard Lua debug library (diagnostic-only `debug.traceback`, `debug.stackdepth`, `debug.where` available via `LuaDebugProvider`)

## Contributing

PRs welcome. Run `go test ./...` before submitting.

## License

MIT
