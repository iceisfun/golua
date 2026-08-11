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
// # Comparing Values
//
// [vm.Value] is deliberately not comparable with Go's ==. Strings are stored
// unboxed as a data pointer plus a length, so == would compare pointers rather
// than contents and report two equal strings as different whenever they were
// built by different paths (a compile-time literal versus a runtime-assembled
// string, say). Value carries a zero-width non-comparable field so that == and
// map[vm.Value]T fail to compile instead of failing silently at run time.
//
// Migration:
//
//	a == b              ->  a.Equal(b)    // content equality, int/float coercion
//	a == vm.Nil         ->  a.IsNil()
//	map[vm.Value]T      ->  map[K]T keyed on an extracted Go value
//	                        (v.AsString(), v.AsInt(), v.AsTable(), ...)
//
// RawEqual is an alias for Equal. Neither runs an __eq metamethod; that
// dispatch belongs to the VM, not to the value.
//
// The compiler cannot catch every case: a Value boxed in an interface{} still
// compiles with ==, and panics at run time with "comparing uncomparable type
// vm.Value". Compare such values by extracting the Value first.
//
// See the examples/ directory for complete usage patterns including sandboxed
// file I/O, Go↔Lua channels, and exposing Go objects to Lua scripts.
package golua
