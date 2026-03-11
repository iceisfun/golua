// Example: Capturing Lua print() output
//
// This example demonstrates using WithCaptureOutput to intercept print() calls
// for testing or processing, instead of writing to stdout.
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
	source := `
		print("hello", "world")
		print(1 + 2)
		print("type is", type(42))

		for i = 1, 3 do
			print("line " .. i)
		end
	`

	// Parse and compile
	block, err := parser.Parse("example", source)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	proto, err := compiler.Compile("example", block)
	if err != nil {
		log.Fatalf("Compile error: %v", err)
	}

	// Create VM with output capture enabled
	v := vm.New(vm.WithCaptureOutput(true))
	stdlib.Open(v)

	// Run the Lua code — print() output goes to the capture buffer, not stdout
	if _, err := v.Run(proto); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	// Inspect captured output
	fmt.Println("=== Captured Output ===")
	for i, line := range v.OutputLines() {
		fmt.Printf("  [%d] %s\n", i, line)
	}

	fmt.Printf("\nLast line: %q\n", v.LastOutput())
	fmt.Printf("Total lines: %d\n", len(v.OutputLines()))

	// Clear the capture buffer when you want to reuse the VM.
	v.ClearOutput()
	fmt.Printf("After clear: %d lines\n", len(v.OutputLines()))
}
