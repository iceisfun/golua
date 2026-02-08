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
- No `io` or `os` library (sandboxed by default)
- No debug library

## Contributing

PRs welcome. Run `go test ./...` before submitting.

## License

MIT
