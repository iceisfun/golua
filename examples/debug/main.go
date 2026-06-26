// Example: Capability-gated debug with a custom LuaDebugProvider
//
// The debug table is built entirely from the provider's Capabilities. By
// returning only the read-only diagnostic flags we expose a debug library
// that has traceback, stackdepth, and where but NOT hooks, local/upvalue
// mutation, or registry/metatable access.
//
// Note: vm.NewDefaultDebugProvider() enables EVERY debug function (it is the
// full standard library). To restrict the surface, implement the one-method
// LuaDebugProvider interface yourself, as diagnosticOnly does below.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// diagnosticOnly is a LuaDebugProvider that exposes only the read-only
// diagnostic functions. Every other debug.* function is left out of the
// table, so scripts cannot install hooks or mutate locals/upvalues.
type diagnosticOnly struct{}

func (diagnosticOnly) Capabilities(ctx context.Context) vm.LuaDebugCaps {
	return vm.LuaDebugCaps{
		AllowTraceback:  true,
		AllowStackDepth: true,
		AllowWhere:      true,
		// All other Allow* flags default to false and stay absent.
	}
}

func main() {
	fmt.Println("=== Diagnostic Debug Provider Example ===")
	fmt.Println("(NOT the standard Lua debug library)")
	fmt.Println()

	source := `
		-- debug.traceback: get a stack trace from nested calls
		local function c()
			return debug.traceback("error in c")
		end
		local function b() return c() end
		local function a() return b() end

		print("--- traceback ---")
		print(a())
		print()

		-- debug.stackdepth: see how deep we are
		print("--- stackdepth ---")
		local function nested()
			return debug.stackdepth()
		end
		print("Top level depth:", debug.stackdepth())
		print("Nested depth:   ", nested())
		print()

		-- debug.where: get source and line at a given level
		print("--- where ---")
		local src, line = debug.where()
		print("Current location:", src .. ":" .. tostring(line))
		print()

		-- These do NOT exist (by design):
		print("--- safety check ---")
		print("debug.sethook:", debug.sethook)
		print("debug.getlocal:", debug.getlocal)
		print("debug.setlocal:", debug.setlocal)
	`

	block, err := parser.Parse("example", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("example", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	v := vm.New()
	if err := v.SetDebugProvider(diagnosticOnly{}); err != nil {
		log.Fatalf("SetDebugProvider: %v", err)
	}
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\n=== Complete ===")
}
