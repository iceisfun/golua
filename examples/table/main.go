// Example: LuaTable interface and deterministic iteration
//
// This example shows how to:
// - Create tables from Go using the LuaTable interface
// - Pass tables to Lua scripts
// - Iterate tables deterministically with next()/pairs()
// - Demonstrate that iteration order is stable across multiple loops
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	// --- Part 1: Go-side table usage via LuaTable interface ---

	fmt.Println("=== Go-side LuaTable usage ===")

	// Create a table through the interface
	var tbl vm.LuaTable = vm.NewEmptyTable()
	tbl.Set(vm.NewString("name"), vm.NewString("golua"))
	tbl.Set(vm.NewString("version"), vm.NewInt(1))
	tbl.Set(vm.NewString("stable"), vm.True)

	// Iterate using Next - order matches insertion
	fmt.Println("Table contents (insertion order):")
	k, v := tbl.Next(vm.Nil)
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v = tbl.Next(k)
	}

	// Iterate again - same order
	fmt.Println("\nSecond iteration (identical order):")
	k, v = tbl.Next(vm.Nil)
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v = tbl.Next(k)
	}

	// Delete a key and iterate
	tbl.Delete(vm.NewString("version"))
	fmt.Println("\nAfter deleting 'version':")
	k, v = tbl.Next(vm.Nil)
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v = tbl.Next(k)
	}

	// --- Part 2: Run the Lua example script ---

	fmt.Println("\n=== Lua-side iteration ===")

	// Find the example.lua file relative to this binary
	scriptPath := filepath.Join("examples", "table", "example.lua")
	source, err := os.ReadFile(scriptPath)
	if err != nil {
		// Try from the current directory
		source, err = os.ReadFile("example.lua")
		if err != nil {
			log.Fatalf("Cannot read example.lua: %v", err)
		}
	}

	block, err := parser.Parse("example.lua", string(source))
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("example.lua", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	luaVM := vm.New()
	stdlib.Open(luaVM)

	_, err = luaVM.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}
