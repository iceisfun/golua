// Example: Calling Lua functions from Go
//
// This example shows how to:
// - Define functions in Lua
// - Call those functions from Go with arguments
// - Get return values back in Go
package main

import (
	"fmt"
	"log"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	// Define some Lua functions
	source := `
		function greet(name)
			return "Hello, " .. name .. "!"
		end

		function add(a, b)
			return a + b
		end

		function getMultiple()
			return 1, 2, 3
		end

		function process(data)
			-- Lua can work with tables passed from Go
			return {
				sum = data.x + data.y,
				product = data.x * data.y
			}
		end
	`

	// Parse and compile
	block, err := parser.Parse("functions", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	proto, err := compiler.Compile("functions", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	// Create VM and run to define the functions
	v := vm.New()
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Now call Lua functions from Go

	// Example 1: Call greet("World")
	greetFn := v.GetGlobal("greet")
	results, err := v.ProtectedCall(greetFn, []vm.Value{vm.NewString("World")})
	if err != nil {
		log.Fatalf("Error calling greet: %v", err)
	}
	fmt.Printf("greet(\"World\") = %s\n", results[0].AsString())

	// Example 2: Call add(10, 20)
	addFn := v.GetGlobal("add")
	results, err = v.ProtectedCall(addFn, []vm.Value{vm.NewInt(10), vm.NewInt(20)})
	if err != nil {
		log.Fatalf("Error calling add: %v", err)
	}
	fmt.Printf("add(10, 20) = %d\n", results[0].AsInt())

	// Example 3: Multiple return values
	getMultipleFn := v.GetGlobal("getMultiple")
	results, err = v.ProtectedCall(getMultipleFn, nil)
	if err != nil {
		log.Fatalf("Error calling getMultiple: %v", err)
	}
	fmt.Printf("getMultiple() = %d, %d, %d\n",
		results[0].AsInt(), results[1].AsInt(), results[2].AsInt())

	// Example 4: Pass a table to Lua and get a table back
	inputTable := vm.NewEmptyTable()
	inputTable.SetString("x", vm.NewInt(5))
	inputTable.SetString("y", vm.NewInt(3))

	processFn := v.GetGlobal("process")
	results, err = v.ProtectedCall(processFn, []vm.Value{vm.NewTable(inputTable)})
	if err != nil {
		log.Fatalf("Error calling process: %v", err)
	}

	resultTable := results[0].AsTable()
	fmt.Printf("process({x=5, y=3}) = {sum=%d, product=%d}\n",
		resultTable.Get(vm.NewString("sum")).AsInt(),
		resultTable.Get(vm.NewString("product")).AsInt())
}
