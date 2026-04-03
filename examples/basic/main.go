// Example: Basic Lua execution
//
// This example shows the simplest way to run Lua code from Go.
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
	// Lua source code
	source := `
		local function factorial(n)
			if n <= 1 then return 1 end
			return n * factorial(n - 1)
		end
		return factorial(10)
	`

	// Parse
	block, err := parser.Parse("factorial", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	// Compile
	proto, err := compiler.Compile("factorial", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	// Create VM with standard library
	v := vm.New()
	stdlib.Open(v)

	// Run
	results, err := v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Print result
	fmt.Printf("factorial(10) = %v\n", results[0].AsInt())
}
