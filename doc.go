// Package golua implements a Lua 5.5 virtual machine in pure Go.
//
// The implementation is split into several sub-packages:
//
//   - [github.com/iceisfun/golua/v2/lexer] — lexical scanner
//   - [github.com/iceisfun/golua/v2/parser] — recursive descent parser producing an AST
//   - [github.com/iceisfun/golua/v2/compiler] — compiler from AST to register-based bytecode
//   - [github.com/iceisfun/golua/v2/vm] — virtual machine, value types, and provider interfaces
//   - [github.com/iceisfun/golua/v2/stdlib] — standard library (string, table, math, io, os, etc.)
//   - [github.com/iceisfun/golua/v2/directives] — source-level parser for `-- @key value` header metadata (golua-specific extension)
//
// Programs embed GoLua by creating a [vm.VM], optionally registering providers
// for I/O, OS, channels, and timing, then loading and running Lua source:
//
//	v := vm.New()
//	stdlib.Open(v)
//	proto, _ := compiler.Compile("main", block)
//	v.Run(proto)
//
// See the examples/ directory for complete usage patterns including sandboxed
// file I/O, Go↔Lua channels, and exposing Go objects to Lua scripts.
package golua
