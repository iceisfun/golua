// Example: Diagnostic debug with DefaultDebugProvider
//
// This is NOT the standard Lua debug library. It exposes only read-only
// diagnostic functions: traceback, stackdepth, and where. No hooks,
// no local/upvalue mutation, no bytecode inspection.
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
	if err := v.SetDebugProvider(vm.NewDefaultDebugProvider()); err != nil {
		log.Fatalf("SetDebugProvider: %v", err)
	}
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	fmt.Println("\n=== Complete ===")
}
