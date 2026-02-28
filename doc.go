// Package golua implements a Lua 5.4 virtual machine in pure Go.
//
// The implementation is split into several sub-packages:
//
//   - [github.com/iceisfun/golua/lexer] — lexical scanner
//   - [github.com/iceisfun/golua/parser] — recursive descent parser producing an AST
//   - [github.com/iceisfun/golua/compiler] — compiler from AST to register-based bytecode
//   - [github.com/iceisfun/golua/vm] — virtual machine, value types, and provider interfaces
//   - [github.com/iceisfun/golua/stdlib] — standard library (string, table, math, io, os, etc.)
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
