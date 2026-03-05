# GoLua — AI Integration Guide

This document provides a complete integration guide for AI systems (LLMs, AI agents, automation tools) that need to embed or interact with GoLua.

## Overview

GoLua is a Lua 5.4 interpreter written in pure Go with zero dependencies. It uses a provider-driven architecture where the host application controls all system access — file I/O, OS operations, module loading, and debug capabilities are opt-in via provider interfaces.

Key properties for AI integration:
- **Sandboxed by default** — no filesystem, network, or OS access unless explicitly granted
- **Deterministic** — per-VM random seeding, no global state leakage between VMs
- **Resource-bounded** — configurable limits on call depth, stack size, instructions, and metatable chains
- **Context-cancelable** — cooperative cancellation via Go contexts
- **Embeddable** — single `go get`, no cgo, no shared libraries

## Embedding GoLua

### Minimal Example

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
    source := `return 2 + 2`
    block, err := parser.Parse("example", source)
    if err != nil {
        panic(err)
    }
    proto, err := compiler.Compile("example", block)
    if err != nil {
        panic(err)
    }

    v := vm.New()
    stdlib.Open(v)

    results, err := v.Run(proto)
    if err != nil {
        panic(err)
    }
    fmt.Println(results[0].AsInt()) // 4
}
```

### With Output Capture

```go
v := vm.New(vm.WithCaptureOutput(true))
stdlib.Open(v)

// ... run Lua code that calls print() ...

lines := v.OutputLines() // []string of all printed lines
last := v.LastOutput()   // most recent line
```

### With Resource Limits

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

## Provider Integration

Providers control what the Lua VM can access. Without providers, modules like `io`, `os`, and `debug` are absent from the Lua environment.

### Code Provider (module loading)

Controls `dofile()`, `loadfile()`, and `require()` file searching:

```go
// Filesystem-based, rooted at a directory
v.SetCodeProvider(vm.NewDirCodeProvider("/path/to/scripts", vm.LuaLoaderCaps{
    AllowDofile:   true,
    AllowLoadfile: true,
}))
```

Or implement the interface for custom sources (databases, embedded assets, HTTP):

```go
type LuaCodeProvider interface {
    LoadChunk(name string, caller *LuaCallerContext) ([]byte, string, error)
    Capabilities() LuaLoaderCaps
}
```

### IO Provider (file operations)

```go
// Read-only, confined to a directory
v.SetIoProvider(vm.NewJailedIoProvider("/allowed/dir"))

// Read-write, confined to a directory
v.SetIoProvider(vm.NewFullIoProvider("/allowed/dir"))
```

### OS Provider

```go
v.SetOsProvider(vm.NewDefaultOsProvider())

// Or filter which env vars are visible
v.SetOsProvider(vm.NewFilteredOsProvider(func(name string) bool {
    return name == "PATH" || name == "HOME"
}))
```

### Debug Provider

```go
v.SetDebugProvider(vm.NewDefaultDebugProvider()) // all capabilities
```

### Print Provider

Route `print()`/`warn()` output to your logging infrastructure:

```go
type LuaPrintProvider interface {
    Print(msg string)
    Warn(msg string)
}
```

## Calling Lua Functions from Go

```go
fn := v.GetGlobal("myFunction")
results, err := v.ProtectedCall(fn, []vm.Value{
    vm.NewInt(42),
    vm.NewString("hello"),
})
```

## Exposing Go Functions to Lua

```go
v.SetGlobal("myFunc", vm.NewNativeFunc(func(v *vm.VM) int {
    arg1 := v.Get(1).AsString()
    v.Set(0, vm.NewString("result: " + arg1))
    return 1 // number of return values
}))
```

## Value Types

| Go Constructor      | Lua Type   | Extraction          |
| -------------------- | ---------- | ------------------- |
| `vm.Nil`             | `nil`      | `.IsNil()`          |
| `vm.NewBool(b)`      | `boolean`  | `.AsBool()`         |
| `vm.NewInt(i)`       | `number`   | `.AsInt()`          |
| `vm.NewFloat(f)`     | `number`   | `.AsFloat()`        |
| `vm.NewString(s)`    | `string`   | `.AsString()`       |
| `vm.NewTable(t)`     | `table`    | `.AsTable()`        |
| `vm.NewNativeFunc(f)` | `function` | `.AsNativeFunc()`  |

Type checking: `.IsInt()`, `.IsFloat()`, `.IsNumber()`, `.IsString()`, `.IsTable()`, `.IsFunction()`, `.IsCallable()`, `.Type()`.

## Typical Integration Patterns

### Game Scripting

```go
// Create a VM per game entity or script
v := vm.New(vm.WithLimits(vm.Limits{
    MaxInstructions: 100000, // bound CPU per frame
    MaxCallDepth:    50,
}))
stdlib.Open(v)

// Expose game API
v.SetGlobal("move", vm.NewNativeFunc(moveFunc))
v.SetGlobal("attack", vm.NewNativeFunc(attackFunc))

// Run behavior script
v.Run(behaviorProto)
```

### Automation / Configuration Pipelines

```go
v := vm.New(vm.WithCaptureOutput(true))
v.SetCodeProvider(vm.NewDirCodeProvider("./configs", vm.LuaLoaderCaps{
    AllowDofile: true,
}))
stdlib.Open(v)

// User configuration scripts can require() other Lua modules
v.Run(configProto)
config := v.GetGlobal("config").AsTable()
```

### Plugin Systems

```go
// Each plugin runs in an isolated VM
for _, plugin := range plugins {
    v := vm.New(vm.WithContext(ctx), vm.WithLimits(limits))
    v.SetPrintProvider(&PluginLogger{Name: plugin.Name})
    stdlib.Open(v)

    // Expose plugin API
    registerPluginAPI(v, plugin)
    v.Run(plugin.Proto)
}
```

### Sandboxed User Input

```go
// Evaluate user-provided expressions safely
v := vm.New(vm.WithLimits(vm.Limits{
    MaxInstructions: 10000,
    MaxCallDepth:    10,
    MaxStackSlots:   1000,
}))
stdlib.Open(v)
// No providers set — no IO, no OS, no file loading
```

## AI-Specific Considerations

### Output Parsing

Use `WithCaptureOutput(true)` and `v.OutputLines()` to capture structured output from Lua scripts without needing file I/O.

### Error Handling

`v.Run()` and `v.ProtectedCall()` return Go errors. Lua `error()` values are preserved through the Go error boundary via `vm.LuaError{Value: Value}`.

### Multiple Isolated VMs

Each `vm.New()` creates a fully independent VM. No state is shared between VMs. This is safe for concurrent execution with proper synchronization on individual VM instances (VMs themselves are not thread-safe).

### Deterministic Execution

- `math.random` is seeded per-VM — different VMs produce different sequences without cross-contamination
- Table iteration order is deterministic (insertion-ordered)
- No ambient global state
