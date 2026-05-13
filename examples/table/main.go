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

	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/stdlib"
	"github.com/iceisfun/golua/v1/vm"
)

func main() {
	// --- Part 1: Go-side table usage via LuaTable interface ---

	fmt.Println("=== Go-side LuaTable usage ===")

	// Create a table through the interface
	tbl := vm.NewEmptyTable()
	if err := tbl.Set(vm.NewString("name"), vm.NewString("golua")); err != nil {
		log.Fatal(err)
	}
	if err := tbl.Set(vm.NewString("version"), vm.NewInt(1)); err != nil {
		log.Fatal(err)
	}
	if err := tbl.Set(vm.NewString("stable"), vm.True); err != nil {
		log.Fatal(err)
	}

	// Iterate using Next - order matches insertion
	fmt.Println("Table contents (insertion order):")
	k, v, err := tbl.Next(vm.Nil)
	if err != nil {
		log.Fatal(err)
	}
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v, err = tbl.Next(k)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Iterate again - same order
	fmt.Println("\nSecond iteration (identical order):")
	k, v, err = tbl.Next(vm.Nil)
	if err != nil {
		log.Fatal(err)
	}
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v, err = tbl.Next(k)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Delete a key and iterate
	if err := tbl.Delete(vm.NewString("version")); err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nAfter deleting 'version':")
	k, v, err = tbl.Next(vm.Nil)
	if err != nil {
		log.Fatal(err)
	}
	for !k.IsNil() {
		fmt.Printf("  %v = %v\n", k, v)
		k, v, err = tbl.Next(k)
		if err != nil {
			log.Fatal(err)
		}
	}

	// --- Part 2: Run the Lua example script ---

	fmt.Println("\n=== Lua-side iteration ===")

	// Find the example.lua file relative to this binary
	scriptPath := filepath.Join("examples", "table", "example.lua")
	source, readErr := os.ReadFile(scriptPath)
	if readErr != nil {
		// Try from the current directory
		source, readErr = os.ReadFile("example.lua")
		if readErr != nil {
			log.Fatalf("Cannot read example.lua: %v", readErr)
		}
	}

	block, parseErr := parser.Parse("example.lua", string(source))
	if parseErr != nil {
		log.Fatalf("Parse error: %v", parseErr)
	}

	proto, compileErr := compiler.Compile("example.lua", block)
	if compileErr != nil {
		log.Fatalf("Compile error: %v", compileErr)
	}

	luaVM := vm.New()
	stdlib.Open(luaVM)

	_, runErr := luaVM.Run(proto)
	if runErr != nil {
		log.Fatalf("Runtime error: %v", runErr)
	}
}
