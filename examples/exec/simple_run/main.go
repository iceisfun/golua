// Example: Simple process execution with exec.run.
//
// Demonstrates running a command and capturing its output.
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
	fmt.Println("=== exec.run Example ===")
	fmt.Println()

	source := `
		local r = exec.run("ls", "-al")
		print("Exit code:", r.code)
		print("Success:", r.success)
		print("Output:")
		print(r.stdout)
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
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	stdlib.Open(v)

	_, err = v.Run(proto)
	if err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}
