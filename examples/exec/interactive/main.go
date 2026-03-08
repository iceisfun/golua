// Example: Interactive process with stdin/stdout.
//
// Demonstrates writing to a process's stdin and reading sorted output.
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
	fmt.Println("=== Interactive Process Example ===")
	fmt.Println()

	source := `
		local p = exec.spawn("sort")

		p:write("banana\n")
		p:write("apple\n")
		p:write("cherry\n")
		p:close_stdin()

		print("Sorted output:")
		for line in p:readlines() do
			print("  " .. line)
		end

		p:wait()
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
