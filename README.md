<p align="center">
  <img src="docs/contrib/logo2.png" alt="GoLua" width="640">
</p>

# GoLua

An embeddable, sandbox-first Lua 5.4 runtime for Go applications, with a small set of experimental 5.5 features. Pure Go, zero dependencies, no cgo.

Good fits include plugin systems, user scripting, game logic, automation, and controlled configuration runtimes.

## Features

- **Full Lua 5.4 language support** (with experimental 5.5 features)
- Coroutines with yield/resume
- Complete metamethod system (`__index`, `__newindex`, `__call`, `__close`, `__eq`, `__lt`, `__le`, `__add`, `__concat`, `__len`, `__tostring`, etc.)
- Module system (`require`, `package.loaded`, `package.preload`, `package.searchers`, `package.searchpath`)
- Pattern matching (`string.match`, `string.gsub`, `string.find`, `string.gmatch`)
- Binary data packing (`string.pack`, `string.unpack`, `string.packsize`)
- Function serialization (`string.dump` / `load` round-trip, Lua 5.4.8 binary format compatible)
- Go-style glob matching (`glob.match`, `glob.match_words`, `glob.match_named`)
- `<const>` and `<close>` variable attributes with to-be-closed support
- Bitwise operators (`&`, `|`, `~`, `<<`, `>>`) and `bit32` compat library
- Integer division (`//`) and full integer/float numeric model
- UTF-8 library (`utf8.char`, `utf8.codepoint`, `utf8.codes`, `utf8.len`, `utf8.offset`)
- Deterministic `math.random` per VM instance (seeding one VM does not affect others)
- Full debug library (`debug.getinfo`, `debug.traceback`, `debug.sethook`, `debug.getlocal`, `debug.setlocal`, `debug.getupvalue`, `debug.setupvalue`, `debug.upvalueid`, `debug.upvaluejoin`)
- Go interop (call Lua from Go, expose Go functions to Lua)
- Sandboxed code loading via [`LuaCodeProvider`](docs/code_provider.md) (controls `dofile`, `loadfile`, and `require`)
- Sandboxed IO via [`LuaIoProvider`](docs/io_provider.md) (includes `JailedIoProvider` for read-only, directory-confined access)
- Sandboxed OS via [`LuaOsProvider`](docs/os_provider.md) (includes `DefaultOsProvider` with optional env filtering)
- Capability-gated channels for Go↔Lua message passing via [`LuaChanProvider`](docs/chan.md)
- Millisecond-precision timing via [`LuaTimeProvider`](docs/time.md) (`time.now`, `time.since`, `time.tick`)
- Optional HTTP client module (`http.get`, `http.post`, `http.fetch`) with automatic JSON coercion
- Modern process execution via [`LuaProcessProvider`](docs/exec.md) (`exec.run`, `exec.spawn` with streaming I/O, stdin, kill, timed waits)
- Output interception via [`LuaPrintProvider`](docs/print_provider.md) (redirect `print()`/`warn()` to logging, per-VM warn isolation)
- Context cancellation and execution limits (call depth, stack, instructions)
- No cgo, no C dependencies, no shared object (.so/.dll) loading
- Single static binary when compiled

## Installation

### Library

```bash
go get github.com/iceisfun/golua
```

### CLI

```bash
go install github.com/iceisfun/golua/cmd/lua@latest
go install github.com/iceisfun/golua/cmd/luac@latest
```

GoLua is pure Go and does not require cgo.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/iceisfun/golua/compiler"
    "github.com/iceisfun/golua/parser"
    "github.com/iceisfun/golua/stdlib"
    "github.com/iceisfun/golua/vm"
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

Core embedding flow: `parser.Parse` -> `compiler.Compile` -> `vm.New` -> `stdlib.Open` -> `v.Run`.

`vm.New()` defaults to `context.Background()`. Use `vm.WithContext(ctx)` to pass a custom context. Call `v.Close(ctx)` when done to shut down any providers that implement `Shutdownable`.

## Develop Locally

```bash
go run ./cmd/lua script.lua
go run ./cmd/lua -e "print(1 + 1)"
go run ./examples/basic
go test ./...
go test ./tests/...
```

## Architecture

GoLua is organized into layered packages with no circular dependencies:

```
Source → Lexer → Parser → AST → Compiler → Proto (bytecode)
                                                ↓
                                    VM (executes bytecode)
                                    ↑           ↑
                                stdlib      Providers
```

| Package       | Purpose                                                                     |
| ------------- | --------------------------------------------------------------------------- |
| `lexer`       | Tokenizes Lua source into a stream of tokens                                |
| `parser`      | Parses tokens into an AST                                                   |
| `ast`         | Abstract syntax tree node definitions                                       |
| `compiler`    | Compiles AST into Lua 5.4 bytecode (`Proto`)                                |
| `vm`          | Executes bytecode, manages stack/coroutines, defines `Value` and `LuaTable` |
| `stdlib`      | Registers standard library functions (`string`, `math`, `table`, etc.)      |
| `check`       | Static diagnostics for editor integration                                   |
| `glob`        | Go-style pattern matching (non-standard extension)                          |
| `stdlib/http` | Optional HTTP client module (non-standard extension)                        |

**Provider interfaces** (`vm` package) control host-system access:

| Interface            | Controls                                      | Implementation                       |
| -------------------- | --------------------------------------------- | ------------------------------------ |
| [`LuaCodeProvider`](docs/code_provider.md)    | `dofile`, `loadfile`, `require` file searcher | `DirCodeProvider`                    |
| [`LuaIoProvider`](docs/io_provider.md)      | `io.*` file operations                        | `JailedIoProvider`, `FullIoProvider` |
| [`LuaOsProvider`](docs/os_provider.md)      | `os.*` core (`clock`, `time`, `date`, `getenv`, `setlocale`) | `DefaultOsProvider`   |
| [`LuaExecProvider`](docs/exec_provider.md)    | `os.execute` command execution                | `DefaultExecProvider`                |
| [`LuaExitHandler`](docs/exit_handler.md)     | `os.exit` VM termination                      | `DefaultExitHandler`                 |
| [`LuaDebugProvider`](docs/debug_provider.md)   | `debug.*` capability gating                   | `DefaultDebugProvider`               |
| [`LuaChanProvider`](docs/chan.md)    | `chan.*` Go↔Lua channels                      | `DefaultChanProvider`                |
| [`LuaTimeProvider`](docs/time.md)    | `time.*` millisecond timing                   | `DefaultTimeProvider`                |
| [`LuaPrintProvider`](docs/print_provider.md)   | `print()`/`warn()` output routing             | `DefaultPrintProvider`               |
| [`LuaProcessProvider`](docs/exec.md) | `exec.*` process spawning and streaming       | `DefaultProcessProvider`             |
| [`LuaLoadLibProvider`](docs/loadlib_provider.md) | `package.loadlib` native module hook          | custom host implementation           |

All provider interface methods receive `ctx context.Context` as their first parameter, carrying the VM's context for cancellation and deadline propagation. Providers may optionally implement `Initializable` (called when set on a VM) or `Shutdownable` (called by `vm.Close(ctx)`) for lifecycle management.

## Examples

See the `examples/` directory for complete examples:

### Start Here

- **[basic](examples/basic/)** - minimal Lua execution from Go
- **[expose_go](examples/expose_go/)** - expose Go functions and tables to Lua
- **[call_lua](examples/call_lua/)** - call Lua functions from Go and read results back
- **[capture_output](examples/capture_output/)** - capture `print()` output in-memory
- **[print_provider](examples/print_provider/)** - route `print()` and `warn()` through host logging

### Sandboxing And Host Integration

- **[code_provider](examples/code_provider/)** - sandboxed file/module loading with `LuaCodeProvider`
- **[jailed_io](examples/jailed_io/)** - confined filesystem and OS access
- **[table](examples/table/)** - `LuaTable` interop and deterministic iteration
- **[chan](examples/chan/)** - Go<->Lua channels with `chan.select`
- **[glob](examples/glob/)** - Go-style pattern matching from Go and Lua

### Advanced And Optional Modules

- **[exec](examples/exec/)** - process execution, streaming I/O, stdin, and timeouts
- **[time](examples/time/)** - millisecond timing with `time.now`, `time.since`, `time.tick`, and `time.once`
- **[http](examples/http/)** - optional HTTP client module with a dedicated example runner
- **[check](examples/check/)** - diagnostics as JSON for editor integrations
- **[editor](examples/editor/)** - Monaco editor with live diagnostics and sandboxed execution
- **[editor_advanced](examples/editor_advanced/)** - browser IDE with completion, hover, diagnostics, and execution
- **[expose_object](examples/expose_object/)** - Go-backed objects with an explicit adapter layer
- **[context](examples/context/)** - context cancellation stops a runaway Lua script

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

Implement [`LuaCodeProvider`](docs/code_provider.md) to control what Lua can load. All provider interface methods take `ctx context.Context` as their first parameter:

```go
type MyProvider struct{}

func (p *MyProvider) LoadChunk(ctx context.Context, name string, caller *vm.LuaCallerContext) ([]byte, string, error) {
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

### package.loadlib Hook

`package.loadlib` is nil by default. Standard C Lua modules (.so/.dll) are compiled
against the PUC-Rio C API (`lua_State*`, `lua_push*`, etc.) and cannot be loaded
directly into GoLua. This provider lets the host implement its own native module
strategy — for example, mapping module names to Go-implemented bindings or using
cgo to bridge platform-specific libraries.

```go
type MyLoadLibProvider struct{}

func (p *MyLoadLibProvider) LoadLib(ctx context.Context, path, init string, caller *vm.LuaCallerContext) (vm.NativeFunc, error) {
    // The host decides how to interpret path and init — they need not
    // refer to actual .so/.dll files. Return an error to deny loading.
    if path == "mylib" && init == "luaopen_mylib" {
        return func(v *vm.VM) int {
            lib := vm.NewEmptyTable()
            lib.SetString("greet", vm.NewNativeFunc(func(v *vm.VM) int {
                v.Set(0, vm.NewString("hello from Go"))
                return 1
            }))
            v.Set(0, vm.NewTable(lib))
            return 1
        }, nil
    }
    return nil, fmt.Errorf("%s: module not available", path)
}

v := vm.New()
v.SetLoadLibProvider(&MyLoadLibProvider{})
stdlib.Open(v)
```

### Sandboxed IO and OS

```go
v := vm.New()

// Read-only file access, confined to a directory
v.SetIoProvider(vm.NewJailedIoProvider("/path/to/allowed/dir"))

// OS functions (clock, time, date, getenv, setlocale)
v.SetOsProvider(vm.NewDefaultOsProvider())

// Command execution (os.execute)
v.SetExecProvider(vm.NewDefaultExecProvider())

// VM termination (os.exit)
v.SetExitHandler(vm.NewDefaultExitHandler())

// Or restrict which env vars are visible
v.SetOsProvider(vm.NewFilteredOsProvider(func(name string) bool {
    return name == "USER" || name == "HOME"
}))

stdlib.Open(v)
```

### Process Execution

The [`exec`](docs/exec.md) module provides modern process control beyond `os.execute`:

```go
v := vm.New()
v.SetProcessProvider(vm.NewDefaultProcessProvider())
stdlib.Open(v)
```

```lua
-- Simple run with captured output
local result = exec.run("ls", "-al")
print(result.stdout)

-- Spawn for streaming and interaction
local p = exec.spawn("sort")
p:write("banana\napple\n")
p:close_stdin()
for line in p:readlines() do print(line) end
p:wait()

-- Shell mode with pipes
local r = exec.run_shell("cat /etc/hosts | grep localhost")
```

See [docs/exec.md](docs/exec.md) for the full API reference.

### Debug Library

The full Lua 5.4 debug library, gated by [`LuaDebugProvider`](docs/debug_provider.md) capabilities:

```go
v := vm.New()
v.SetDebugProvider(vm.NewDefaultDebugProvider()) // all capabilities enabled
stdlib.Open(v)
// Lua now has: debug.getinfo, debug.traceback, debug.sethook, debug.gethook,
// debug.getlocal, debug.setlocal, debug.getupvalue, debug.setupvalue,
// debug.upvalueid, debug.upvaluejoin, debug.getmetatable, debug.setmetatable,
// debug.getregistry
```

Individual capabilities can be enabled/disabled via [`LuaDebugCaps`](docs/debug_provider.md) fields (e.g. `AllowSetHook`, `AllowSetLocal`, `AllowUpvalueJoin`).

### Print Interception

Route `print()` and `warn()` output through your logging infrastructure:

```go
type LoggingProvider struct{ Name string }

func (p *LoggingProvider) Print(ctx context.Context, msg string) {
    log.Printf("[%s] %s", p.Name, msg)
}
func (p *LoggingProvider) Warn(ctx context.Context, msg string) {
    log.Printf("[%s] WARN: %s", p.Name, msg)
}

v := vm.New()
v.SetPrintProvider(&LoggingProvider{Name: "inventory.lua"})
stdlib.Open(v)
```

Without a provider, `print()` writes to stdout (or the `WithCaptureOutput` buffer) and `warn()` writes to stderr — matching standard Lua behavior.

The `warn("@on")`/`warn("@off")` flag is per-VM, so disabling warnings in one VM does not affect others.

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

| Function                          | Description                                                              |
| --------------------------------- | ------------------------------------------------------------------------ |
| `chan.make(size?)`                | Create a new channel (0 = unbuffered)                                    |
| `chan.select(ch1, ..., timeout?)` | Receive from any ready channel; returns `idx, val, ok`                   |
| `ch:send(val)`                    | Blocking send (panics on interrupt or closed channel)                    |
| `ch:recv()`                       | Blocking receive; returns `val, ok` (`ok=false` when closed and drained) |
| `ch:close()`                      | Close the channel (panics if already closed)                             |
| `ch:try_send(val)`                | Non-blocking send; returns `bool`                                        |
| `ch:try_recv()`                   | Non-blocking receive; returns `val, ok, received`                        |

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

-- one-time initialization guard
if time.once() then load_resources() end
```

| Function                | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| `time.now()`            | Current time in milliseconds (integer)                                            |
| `time.since(t)`         | Milliseconds elapsed since `t`                                                    |
| `time.tick([name,] ms)` | Returns `true` once per `ms` interval, `false` otherwise                          |
| `time.once([name])`     | Returns `true` on the first call for a given key, `false` on all subsequent calls |

`time.tick` and `time.once` are **GoLua extensions** (not part of standard Lua). When `name` is omitted, both functions auto-key by callsite — the VM inspects the calling function's source file and line number (`source:line`) so each call location gets independent state. Pass an explicit `name` string to share state across call locations. The `time` table is **absent by default** and only appears when the host sets a `LuaTimeProvider`.

### LuaTable Interface

Tables implement the `LuaTable` interface, which is the contract used by the VM and stdlib:

```go
type LuaTable interface {
    Get(key Value) Value
    Set(key Value, val Value) error
    Delete(key Value) error
    Next(key Value) (nextKey Value, val Value, err error)
    Len() int
    Metatable() LuaTable
    SetMetatable(mt LuaTable)
    IsThread() bool  // Whether this table represents a coroutine thread
    VMRef() *VM      // Coroutine VM reference, or nil
}
```

`LuaTable.Get()` performs **raw access** (equivalent to Lua's `rawget()`) — it does not invoke `__index` metamethods. To get Lua-style indexing that walks the `__index` chain, use `v.TableGet(tbl, key)` or `v.TableGetInt(tbl, n)` instead. Similarly, `v.SetIndexValue()` respects `__newindex`. This distinction matters when reading from class instances created via `setmetatable`.

The default `*Table` implementation uses an ordered keys slice for the hash part, so `next()`/`pairs()` iteration is deterministic (insertion-ordered). This avoids the non-deterministic `range map` behavior in Go. Mutation during iteration is not safe; inserting or deleting keys may skip entries or produce duplicates. Tables have no implicit thread safety -- concurrent read+write requires external synchronization.

## Standard Library

`stdlib.Open(v)` registers all standard modules. Capability-gated modules only appear when their provider is set.

Standard Lua modules are available by default. GoLua-specific extensions and host-facing capabilities include `glob`, `chan`, `time`, `exec`, and the separate `stdlib/http` module.

| Module      | Requires Provider    | Description                                                                                   |
| ----------- | -------------------- | --------------------------------------------------------------------------------------------- |
| `string`    | No                   | Pattern matching, formatting, byte manipulation, `pack`/`unpack`, `dump`                      |
| `math`      | No                   | Math functions with per-VM deterministic random                                               |
| `table`     | No                   | Table manipulation (sort, concat, insert, remove, move, pack, unpack)                         |
| `coroutine` | No                   | Coroutine creation and control                                                                |
| `utf8`      | No                   | UTF-8 encoding/decoding (strict mode)                                                         |
| `bit32`     | No                   | Lua 5.2 bitwise compat library                                                                |
| `glob`      | No                   | Case-insensitive Go-style pattern matching (`match`, `match_words`, `match_named`)            |
| `io`        | [`LuaIoProvider`](docs/io_provider.md)      | File I/O (absent by default)                                                                  |
| `os`        | [`LuaOsProvider`](docs/os_provider.md)      | OS core: `clock`, `time`, `date`, `difftime`, `getenv`, `setlocale` (`execute`/`exit`/`rename` also require their own providers) |
| `package`   | No*                  | Module system: `require`, `package.loaded`, `package.preload`, `package.searchers`, `searchpath` |
| `debug`     | [`LuaDebugProvider`](docs/debug_provider.md)   | Full Lua 5.4 debug library: getinfo, hooks, locals, upvalues (absent by default)              |
| `chan`      | [`LuaChanProvider`](docs/chan.md)    | Go↔Lua message passing channels (absent by default)                                           |
| `time`      | [`LuaTimeProvider`](docs/time.md)    | Millisecond timing: now, since, periodic tick (absent by default)                             |
| `exec`      | [`LuaProcessProvider`](docs/exec.md) | Process execution: run, spawn, streaming I/O, stdin, kill ([docs](docs/exec.md))              |
| `http`      | Separate module      | HTTP client: get, post, put, patch, delete, fetch ([docs](docs/http.md))                      |

\* `package.loadlib` is nil by default; set `LuaLoadLibProvider` to provide host-defined native module loading.

## Security Model

GoLua is sandboxed by default. The VM starts with no access to the host system. Capabilities are granted explicitly by the host via providers.

A fresh `vm.New()` instance has no file, network, process, or debug access unless the host opts in.

```
┌───────────────────────────────────────────────────────────────────────────────────────┐
│                                    Host (Go)                                          │
│                                                                                       │
│  ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌─────────────┐ ┌────────────┐ ┌──────────┐ │
│  │CodeProvider│ │IoProvider│ │OsProvider│ │DebugProvider│ │ChanProvider│ │  Time    │ │
│  │ (optional) │ │(optional)│ │(optional)│ │ (optional)  │ │ (optional) │ │(optional)│ │
│  │            │ │          │ │ExecProv. │ │             │ │            │ │          │ │
│  │            │ │          │ │ExitHndlr │ │             │ │            │ │          │ │
│  │            │ │          │ │ProcessP. │ │             │ │            │ │          │ │
│  └─────┬──────┘ └────┬─────┘ └────┬─────┘ └──────┬──────┘ └─────┬──────┘ └────┬─────┘ │
│        │             │            │              │              │             │       │
│  ┌─────▼─────────────▼────────────▼──────────────▼──────────────▼─────────────▼──┐    │
│  │                              VM  (sandbox)                                    │    │
│  │                                                                               │    │
│  │  string, math, table, coroutine, utf8, bit32, glob, package                   │    │
│  │  io*, os*, debug*, chan*, time*, exec*           (* = provider-gated)         │    │
│  │  require → CodeProvider (Lua file searcher)                                   │    │
│  │  print/warn → PrintProvider (or stdout/stderr fallback)                       │    │
│  └───────────────────────────────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

- No filesystem access unless explicitly provided
- No OS access unless explicitly provided
- No native code loading (C Lua modules are incompatible; host can provide Go-native bindings via [`LuaLoadLibProvider`](docs/loadlib_provider.md))
- No ambient authority

The default IO provider ([`JailedIoProvider`](docs/io_provider.md)) enforces:

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
├── check/         # Lua source diagnostics for editor integration
├── compiler/      # Bytecode compiler
├── docs/          # Additional documentation
├── glob/          # Go-style glob matching package, diverges from standard lua
├── lexer/         # Lexical analyzer
├── parser/        # Lua parser
├── stdlib/        # Standard library implementation
├── token/         # Token definitions
├── vm/            # Virtual machine
├── tests/         # Lua test files
├── examples/      # Usage examples
└── cmd/
    ├── ast/       # AST printer/debugger CLI
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

# Run with test mode (enables debug provider, jailed IO, code provider)
go run ./cmd/lua --test script.lua

# Compile and show bytecode
go run ./cmd/luac script.lua
```

## Limitations

Current compatibility notes and intentional boundaries:

- The CLI enables a practical host-facing set of providers by default: `DefaultOsProvider`, `FullIoProvider`, `DirCodeProvider`, `DefaultExecProvider`, `DefaultExitHandler`, and `DefaultDebugProvider`. `--test` switches to a more test-like environment with `JailedIoProvider`, `DirCodeProvider`, and `DefaultDebugProvider`, without enabling `os.execute` or `os.exit`.
- No C module loading — standard C Lua modules (.so/.dll) are compiled against the PUC-Rio C API and are not compatible with GoLua's VM. `package.loadlib` is nil by default; setting a [`LuaLoadLibProvider`](docs/loadlib_provider.md) lets the host provide Go-native bindings or cgo-bridged libraries under the same API surface. `require` loads Lua modules via `LuaCodeProvider`.
- No `io.stdin`/`io.stdout`/`io.stderr` in the library by default (the CLI at `cmd/lua` provides full stdio via its environment, but `vm.New()` does not to maintain the sandbox)
- No `io.write` in `JailedIoProvider` (read-only by design; use `FullIoProvider` for read-write access)
- Binary chunk format is compatible with Lua 5.4.8 — `load(string.dump(f))` round-tripping works, and chunks dumped by GoLua can be loaded by reference Lua 5.4.8 and vice versa. However, bytecode details may differ between compilers.
- `os.setlocale` only supports the `"C"` locale — Go has no native locale support. Queries return `"C"` and setting any other locale returns `nil`.
- GC behavior differs from C Lua — GoLua delegates garbage collection entirely to Go's runtime GC. `collectgarbage("collect")` triggers `runtime.GC()` but Go's GC timing is non-deterministic, so tests that depend on exact finalization order or count may not pass.
- VM instances are isolated but not safe for concurrent mutation from multiple goroutines without external synchronization.

## Contributing

PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. Run `go test ./...` before submitting.

## License

MIT
